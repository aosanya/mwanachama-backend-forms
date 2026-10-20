package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestAnAnswerGivenBeforeSendsWereRecordedCountsAsASendOnceProvisioned(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Earlier"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	s := testSpec(t, "test")
	o, _ := s.ByRole("submission")
	if err := db.Exec("delete from " + s.TableFor(o)).Error; err != nil {
		t.Fatalf("forget the send: %v", err)
	}
	if rows, _ := mgr.ListAnswered(ctx, "alice"); len(rows) != 0 {
		t.Fatalf("with the send forgotten, answered = %+v", rows)
	}

	for range 2 {
		if err := mwanachamaforms.Provision(db, s); err != nil {
			t.Fatalf("Provision: %v", err)
		}
	}
	got, err := mgr.ListSubmissions(ctx, f.ID, "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	answers, _ := mgr.ListAnswers(ctx, f.ID, "alice")
	if len(got) != 1 || got[0].MemberID != "alice" || got[0].Answers != 1 ||
		got[0].SubmittedAt != answers[0].AnsweredAt || got[0].ChapterID != "ward-1" {
		t.Fatalf("recorded sends = %+v, want alice's one, from her answer", got)
	}
}

func TestAnEmptySendIsStillRefusedWhereAnAnswerIsRequired(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Required", ClosesAt: futureRFC3339(time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "Why?", Required: true,
	}}); err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := mgr.Publish(ctx, f.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", nil); !errors.Is(err, mwanachamaforms.ErrInvalidAnswer) {
		t.Fatalf("empty send on a required form: %v, want ErrInvalidAnswer", err)
	}
	if _, err := mgr.GetSubmission(ctx, f.ID, "alice"); !errors.Is(err, mwanachamaforms.ErrSubmissionNotFound) {
		t.Fatalf("a refused send was recorded: %v", err)
	}
}
