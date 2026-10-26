package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func composeStanding(t *testing.T, mgr mwanachamaforms.FormManager, landing mwanachamaforms.Landing) (mwanachamaforms.Form, mwanachamaforms.Question) {
	t.Helper()
	return composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Request", Standing: true, Landing: landing})
}

func TestALandingIsKeptThroughCreateReadAndList(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	for _, landing := range []mwanachamaforms.Landing{mwanachamaforms.LandingThread, mwanachamaforms.LandingCase, mwanachamaforms.LandingNone} {
		f, err := mgr.Create(ctx, mwanachamaforms.Form{
			Title: "Request", OriginatorChapterID: "council-" + string(landing),
			ClosesAt: futureRFC3339(time.Hour), Standing: true, Landing: landing,
		})
		if err != nil {
			t.Fatalf("Create %q: %v", landing, err)
		}
		if f.Landing != landing {
			t.Fatalf("Create returned landing %q, want %q", f.Landing, landing)
		}
		got, err := mgr.Get(ctx, f.ID)
		if err != nil || got.Landing != landing {
			t.Fatalf("Get landing = %q, %v, want %q", got.Landing, err, landing)
		}
		listed, err := mgr.ListForChapter(ctx, "council-"+string(landing))
		if err != nil || len(listed) != 1 || listed[0].Landing != landing {
			t.Fatalf("ListForChapter = %+v, %v, want one form landing %q", listed, err, landing)
		}
	}
}

func TestALandingSurvivesPublishing(t *testing.T) {
	mgr := newTestManager(t)
	f, _ := composeStanding(t, mgr, mwanachamaforms.LandingCase)
	if f.Landing != mwanachamaforms.LandingCase {
		t.Fatalf("published landing = %q, want case", f.Landing)
	}
	got, err := mgr.Get(context.Background(), f.ID)
	if err != nil || got.Landing != mwanachamaforms.LandingCase || !got.Standing {
		t.Fatalf("Get = %+v, %v", got, err)
	}
}

func TestALandingOnAFormThatIsNotStandingIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	for _, landing := range []mwanachamaforms.Landing{mwanachamaforms.LandingThread, mwanachamaforms.LandingCase} {
		_, err := mgr.Create(context.Background(), mwanachamaforms.Form{
			Title: "Request", OriginatorChapterID: "council", ClosesAt: futureRFC3339(time.Hour), Landing: landing,
		})
		if !errors.Is(err, mwanachamaforms.ErrInvalidForm) {
			t.Fatalf("landing %q without standing: err = %v, want ErrInvalidForm", landing, err)
		}
	}
}

func TestAnUnknownLandingIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	for _, standing := range []bool{true, false} {
		_, err := mgr.Create(context.Background(), mwanachamaforms.Form{
			Title: "Request", OriginatorChapterID: "council", ClosesAt: futureRFC3339(time.Hour),
			Standing: standing, Landing: "email",
		})
		if !errors.Is(err, mwanachamaforms.ErrInvalidForm) {
			t.Fatalf("unknown landing, standing=%v: err = %v, want ErrInvalidForm", standing, err)
		}
	}
}

func TestAnEditCannotChangeOrClearALanding(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{
		Title: "Request", OriginatorChapterID: "council", ClosesAt: futureRFC3339(time.Hour),
		Standing: true, Landing: mwanachamaforms.LandingThread,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, attempt := range []mwanachamaforms.Landing{mwanachamaforms.LandingNone, mwanachamaforms.LandingCase} {
		cur, _ := mgr.Get(ctx, f.ID)
		cur.Landing = attempt
		if _, err := mgr.Update(ctx, cur); err != nil {
			t.Fatalf("Update naming landing %q: %v", attempt, err)
		}
		if got, _ := mgr.Get(ctx, f.ID); got.Landing != mwanachamaforms.LandingThread {
			t.Fatalf("an edit naming landing %q left %q, want thread", attempt, got.Landing)
		}
	}
}

func TestAnEditCannotGiveAFormALanding(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f := newDraftForm(t, mgr)
	f.Landing = mwanachamaforms.LandingCase
	f.Standing = true
	if _, err := mgr.Update(ctx, f); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := mgr.Get(ctx, f.ID)
	if got.Landing != mwanachamaforms.LandingNone || got.Standing {
		t.Fatalf("an edit gave the form standing %v landing %q", got.Standing, got.Landing)
	}
}

func TestAStandingFormIsStoredAdvisoryWhateverWasSent(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	standing, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Issues", ClosesAt: futureRFC3339(time.Hour), Standing: true, WindowAdvisory: false})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ordinary, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Survey", ClosesAt: futureRFC3339(time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !standing.WindowAdvisory {
		t.Fatalf("Create returned a standing form that keeps its window: %+v", standing)
	}
	if got, _ := mgr.Get(ctx, standing.ID); !got.WindowAdvisory {
		t.Fatalf("a standing form was stored keeping its window: %+v", got)
	}
	if got, _ := mgr.Get(ctx, ordinary.ID); got.WindowAdvisory {
		t.Fatalf("a form that is not standing was stored advisory: %+v", got)
	}
}

func TestAStandingFormPastItsClosingTimeStillTakesASend(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	f, q := composeStanding(t, mgr, mwanachamaforms.LandingThread)
	moveClosesAtIntoThePast(t, db, f.ID)
	for range 2 {
		if err := submitOne(mgr, f.ID, q); err != nil {
			t.Fatalf("send to a standing form past its closes_at: %v", err)
		}
	}
	rows, err := mgr.ListAnswered(context.Background(), "alice")
	if err != nil || len(rows) != 2 || !rows[0].Open || !rows[1].Open {
		t.Fatalf("answered = %+v, %v, want two sends of a form still open", rows, err)
	}
}

func TestAStandingFormBeforeItsOpeningTimeStillTakesASend(t *testing.T) {
	mgr := newTestManager(t)
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Early", Standing: true, OpensAt: futureRFC3339(time.Hour)})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("send before opens_at: %v", err)
	}
}

func TestAFormThatIsNotStandingStillKeepsItsWindow(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Survey"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("inside the window: %v", err)
	}
	moveClosesAtIntoThePast(t, db, f.ID)
	if err := submitOne(mgr, f.ID, q); !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("err = %v, want ErrOutsideWindow", err)
	}
}

func TestAStandingFormComposedWithAScheduleStillTakesRepeatedSends(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly request", Standing: true, Schedule: weeklySchedule()})
	if !f.Schedule.Repeats() {
		t.Fatalf("the schedule was not kept: %+v", f.Schedule)
	}
	for _, text := range []string{"first", "second", "third"} {
		if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: text}}); err != nil {
			t.Fatalf("SubmitAnswers %q: %v", text, err)
		}
	}
	sends, err := mgr.ListSubmissions(ctx, f.ID, "alice", "")
	if err != nil || len(sends) != 3 {
		t.Fatalf("sends = %+v, %v, want three kept apart", sends, err)
	}
	keys := map[string]bool{}
	for _, s := range sends {
		if s.PeriodKey == "" || s.PeriodKey == currentWeeklyKey(t) || keys[s.PeriodKey] {
			t.Fatalf("send %+v took the schedule's period instead of its own key", s)
		}
		keys[s.PeriodKey] = true
	}
}

func TestASendOnAStandingFormReturnsTheIdOfItsOwnRecord(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeStanding(t, mgr, mwanachamaforms.LandingCase)
	seen := map[string]bool{}
	for _, text := range []string{"broken pump", "leaking roof"} {
		stored, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: text}})
		if err != nil || len(stored) != 1 {
			t.Fatalf("SubmitAnswers %q: %v %+v", text, err, stored)
		}
		id := stored[0].SendID
		if id == "" || seen[id] {
			t.Fatalf("send id %q is empty or already used by an earlier send", id)
		}
		seen[id] = true
		send, err := mgr.GetSend(ctx, f.ID, id, "alice")
		if err != nil {
			t.Fatalf("GetSend %q: %v", id, err)
		}
		if send.Submission.ID != id || send.Submission.PeriodKey != stored[0].PeriodKey || len(send.Answers) != 1 || send.Answers[0].ValueText != text {
			t.Fatalf("GetSend(%q) = %+v, want the send just made", id, send)
		}
	}
}

func TestTheSendIdIsTheOneTheAnsweredListAndTheSubmissionsCarry(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeStanding(t, mgr, mwanachamaforms.LandingThread)
	stored, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "need a ride"}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	held, err := mgr.ListSubmissions(ctx, f.ID, "alice", stored[0].PeriodKey)
	if err != nil || len(held) != 1 || held[0].ID != stored[0].SendID {
		t.Fatalf("submissions = %+v, %v, want one with id %q", held, err, stored[0].SendID)
	}
	answered, err := mgr.ListAnswered(ctx, "alice")
	if err != nil || len(answered) != 1 || answered[0].SendID != stored[0].SendID {
		t.Fatalf("answered = %+v, %v, want send id %q", answered, err, stored[0].SendID)
	}
}

func TestAResendToAnOrdinaryFormNamesTheSameSendEachTime(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Survey"})
	first, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "first"}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	second, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "second"}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	if first[0].SendID == "" || first[0].SendID != second[0].SendID {
		t.Fatalf("send ids %q and %q, want one send edited", first[0].SendID, second[0].SendID)
	}
}

func TestTheSendIdIsNeverStoredOnAnAnswer(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeStanding(t, mgr, mwanachamaforms.LandingThread)
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "x", SendID: "forged"}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	read, err := mgr.ListAllAnswers(ctx, f.ID, "alice", "")
	if err != nil || len(read) != 1 || read[0].SendID != "" {
		t.Fatalf("answers = %+v, %v, want the send id left off a stored answer", read, err)
	}
}

func TestProvisionMakesAnEarlierStandingFormAdvisory(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	s := testSpec(t, "test")
	o, _ := s.ByRole("form")
	table := s.TableFor(o)

	standing, q := composeStanding(t, mgr, mwanachamaforms.LandingNone)
	ordinary, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Survey"})
	if err := db.Table(table).Where("id = ?", standing.ID).Update("window_advisory", false).Error; err != nil {
		t.Fatalf("make the stored form keep its window: %v", err)
	}
	moveClosesAtIntoThePast(t, db, standing.ID)
	if err := submitOne(mgr, standing.ID, q); !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("before provisioning: err = %v, want ErrOutsideWindow", err)
	}
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if err := submitOne(mgr, standing.ID, q); err != nil {
		t.Fatalf("after provisioning: %v", err)
	}
	if got, _ := mgr.Get(ctx, ordinary.ID); got.WindowAdvisory {
		t.Fatalf("provisioning made a form that is not standing advisory: %+v", got)
	}
}

func TestAFormThatPredatesTheLandingColumnReadsBackWithNone(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	s := testSpec(t, "test")
	o, _ := s.ByRole("form")
	table := s.TableFor(o)

	f, _ := composeStanding(t, mgr, mwanachamaforms.LandingNone)
	if err := db.Exec("alter table " + table + " drop column landing").Error; err != nil {
		t.Fatalf("drop landing: %v", err)
	}
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision over the older table: %v", err)
	}
	got, err := mgr.Get(context.Background(), f.ID)
	if err != nil || got.Landing != mwanachamaforms.LandingNone || !got.Standing {
		t.Fatalf("Get = %+v, %v, want a standing form landing nowhere", got, err)
	}
}

func TestAnEmptySendToAStandingFormReturnsNoAnswersAndIsTheNewestSubmission(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeStanding(t, mgr, mwanachamaforms.LandingCase)
	first, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "with an answer"}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	empty, err := mgr.SubmitAnswers(ctx, f.ID, "alice", nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty send = %+v, %v, want no answers", empty, err)
	}
	held, err := mgr.ListSubmissions(ctx, f.ID, "alice", "")
	if err != nil || len(held) != 2 {
		t.Fatalf("submissions = %+v, %v, want two", held, err)
	}
	if held[0].ID != first[0].SendID || held[1].ID == first[0].SendID || held[1].Answers != 0 {
		t.Fatalf("submissions = %+v, want the empty send last, apart from %q", held, first[0].SendID)
	}
}
