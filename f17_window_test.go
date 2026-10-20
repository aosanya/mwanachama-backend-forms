package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func composeAimedForm(t *testing.T, mgr mwanachamaforms.FormManager, f mwanachamaforms.Form) (mwanachamaforms.Form, mwanachamaforms.Question) {
	t.Helper()
	ctx := context.Background()
	f.OriginatorChapterID = "council"
	if f.ClosesAt == "" {
		f.ClosesAt = futureRFC3339(24 * time.Hour)
	}
	created, err := mgr.Create(ctx, f)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: created.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is the ward?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := mgr.AddTarget(ctx, mwanachamaforms.Target{FormID: created.ID, ChapterID: "ward-1"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	published, _, err := mgr.Publish(ctx, created.ID, "", "sender-1", "")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return published, q
}

func moveClosesAtIntoThePast(t *testing.T, db *gorm.DB, formID string) {
	t.Helper()
	s := testSpec(t, "test")
	o, _ := s.ByRole("form")
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	if err := db.Table(s.TableFor(o)).Where("id = ?", formID).Update("closes_at", past).Error; err != nil {
		t.Fatalf("move closes_at: %v", err)
	}
}

func submitOne(mgr mwanachamaforms.FormManager, formID string, q mwanachamaforms.Question) error {
	_, err := mgr.SubmitAnswers(context.Background(), formID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine"}})
	return err
}

func waitingTitles(t *testing.T, mgr mwanachamaforms.FormManager) map[string]bool {
	t.Helper()
	rows, err := mgr.ListWaiting(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListWaiting: %v", err)
	}
	out := map[string]bool{}
	for _, r := range rows {
		out[r.Title] = true
	}
	return out
}

func aliceInWard1(mgr mwanachamaforms.FormManager) {
	mgr.UseGroups(func(context.Context, string) ([]string, error) { return []string{"ward-1"}, nil })
}

func TestANewFormKeepsItsWindowUnlessItSaysOtherwise(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	kept, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Kept", ClosesAt: futureRFC3339(time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	advisory, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Advisory", ClosesAt: futureRFC3339(time.Hour), WindowAdvisory: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for id, want := range map[string]bool{kept.ID: false, advisory.ID: true} {
		got, err := mgr.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.WindowAdvisory != want {
			t.Fatalf("form %s window_advisory = %v, want %v", id, got.WindowAdvisory, want)
		}
	}
}

func TestEditingADraftNeverChangesWhetherItsWindowIsKept(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Kept", ClosesAt: futureRFC3339(time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	f.WindowAdvisory = true
	if _, err := mgr.Update(ctx, f); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := mgr.Get(ctx, f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.WindowAdvisory {
		t.Fatal("an edit turned the window advisory; it is chosen when the form is composed")
	}
}

func TestAKeptWindowRefusesAnswersBeforeItOpens(t *testing.T) {
	mgr := newTestManager(t)
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Early", OpensAt: futureRFC3339(time.Hour)})
	if err := submitOne(mgr, f.ID, q); !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("err = %v, want ErrOutsideWindow", err)
	}
}

func TestAKeptWindowRefusesAnswersAfterItCloses(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Late"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("inside the window: %v", err)
	}
	moveClosesAtIntoThePast(t, db, f.ID)
	if err := submitOne(mgr, f.ID, q); !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("err = %v, want ErrOutsideWindow", err)
	}
}

func TestAnAdvisoryWindowLeavesStatusToDecide(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	early, eq := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Early", OpensAt: futureRFC3339(time.Hour), WindowAdvisory: true})
	late, lq := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Late", WindowAdvisory: true})
	moveClosesAtIntoThePast(t, db, late.ID)

	if err := submitOne(mgr, early.ID, eq); err != nil {
		t.Fatalf("before an advisory window opens: %v", err)
	}
	if err := submitOne(mgr, late.ID, lq); err != nil {
		t.Fatalf("after an advisory window closes: %v", err)
	}
	if _, err := mgr.Close(context.Background(), late.ID, "sender-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := submitOne(mgr, late.ID, lq); !errors.Is(err, mwanachamaforms.ErrFormNotOpen) {
		t.Fatalf("a closed advisory form: err = %v, want ErrFormNotOpen", err)
	}
}

func TestTheWaitingListDropsAFormOutsideAKeptWindowOnly(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	aliceInWard1(mgr)
	composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Current"})
	composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "NotYet", OpensAt: futureRFC3339(time.Hour)})
	past, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Over"})
	composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "NotYetAdvisory", OpensAt: futureRFC3339(time.Hour), WindowAdvisory: true})
	pastAdvisory, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "OverAdvisory", WindowAdvisory: true})
	moveClosesAtIntoThePast(t, db, past.ID)
	moveClosesAtIntoThePast(t, db, pastAdvisory.ID)

	got := waitingTitles(t, mgr)
	want := map[string]bool{"Current": true, "NotYetAdvisory": true, "OverAdvisory": true}
	if len(got) != len(want) {
		t.Fatalf("waiting = %v, want %v", got, want)
	}
	for title := range want {
		if !got[title] {
			t.Fatalf("waiting = %v, want %v", got, want)
		}
	}
}

func TestAFormThatPredatesTheFieldKeepsItsWindow(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	aliceInWard1(mgr)
	s := testSpec(t, "test")
	o, _ := s.ByRole("form")
	table := s.TableFor(o)

	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Older"})
	if err := db.Exec("alter table " + table + " drop column window_advisory").Error; err != nil {
		t.Fatalf("drop window_advisory: %v", err)
	}
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision over the older table: %v", err)
	}
	moveClosesAtIntoThePast(t, db, f.ID)

	got, err := mgr.Get(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.WindowAdvisory {
		t.Fatal("a form written before the column existed read back advisory")
	}
	if err := submitOne(mgr, f.ID, q); !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("err = %v, want ErrOutsideWindow", err)
	}
	if waitingTitles(t, mgr)["Older"] {
		t.Fatal("a form past its closes_at, written before the column existed, is still waiting")
	}
}
