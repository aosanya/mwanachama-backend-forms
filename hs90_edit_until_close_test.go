package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestAnEditBeforeCloseChangesTheRecordAndKeepsItsFirstTimeAndGroup(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Ward check"})

	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine"}}); err != nil {
		t.Fatalf("first send: %v", err)
	}
	first, err := mgr.ListAnswers(ctx, f.ID, "alice")
	if err != nil || len(first) != 1 || first[0].EditedAt != "" {
		t.Fatalf("first send = %+v, %v, want one unedited answer", first, err)
	}

	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "better"}}); err != nil {
		t.Fatalf("edit before close: %v", err)
	}
	got, err := mgr.ListAnswers(ctx, f.ID, "alice")
	if err != nil || len(got) != 1 {
		t.Fatalf("after the edit = %+v, %v, want still one record", got, err)
	}
	if got[0].ID != first[0].ID || got[0].ValueText != "better" || got[0].EditedAt == "" ||
		got[0].AnsweredAt != first[0].AnsweredAt || got[0].ChapterID != first[0].ChapterID {
		t.Fatalf("the edit did not change the record in place: before %+v, after %+v", first[0], got[0])
	}
	s, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || s.EditedAt == "" || s.SubmittedAt != first[0].AnsweredAt || s.Answers != 1 {
		t.Fatalf("submission = %+v, %v, want edited once and still sent at the first time", s, err)
	}
}

func TestAnEditAfterCloseIsRefusedAndLeavesTheRecordAsSent(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Ward check"})

	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine"}}); err != nil {
		t.Fatalf("first send: %v", err)
	}
	moveClosesAtIntoThePast(t, db, f.ID)

	_, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "too late"}})
	if !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("edit after close: err = %v, want ErrOutsideWindow", err)
	}
	got, err := mgr.ListAnswers(ctx, f.ID, "alice")
	if err != nil || len(got) != 1 || got[0].ValueText != "fine" || got[0].EditedAt != "" {
		t.Fatalf("after the refused edit = %+v, %v, want the answer as first sent", got, err)
	}
	s, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || s.EditedAt != "" {
		t.Fatalf("submission = %+v, %v, want it unedited", s, err)
	}
}

func TestAnEditToAFormClosedByItsSenderIsRefusedBeforeItsDate(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Ward check"})

	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine"}}); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if _, err := mgr.Close(ctx, f.ID, "sender-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "again"}})
	if !errors.Is(err, mwanachamaforms.ErrFormNotOpen) {
		t.Fatalf("edit to a closed form: err = %v, want ErrFormNotOpen", err)
	}
}
