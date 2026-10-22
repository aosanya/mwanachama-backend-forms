package mwanachamaforms_test

import (
	"context"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestF41ASendRecordsTheLanguageItWasAnsweredInAndAResendItsOwn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Spoken"})
	answers := []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine"}}
	if _, err := mgr.SubmitAnswersIn(ctx, f.ID, "alice", "", "sw", answers); err != nil {
		t.Fatalf("SubmitAnswersIn: %v", err)
	}
	first, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if first.Language != "sw" {
		t.Fatalf("first send language = %q, want sw", first.Language)
	}
	if _, err := mgr.SubmitAnswersIn(ctx, f.ID, "alice", "", "en", answers); err != nil {
		t.Fatalf("resend: %v", err)
	}
	again, _ := mgr.GetSubmission(ctx, f.ID, "alice")
	if again.Language != "en" || again.SubmittedAt != first.SubmittedAt {
		t.Fatalf("resend = %+v, want language en and the first send time kept", again)
	}
}

func TestF41ASendNamingNoLanguageRecordsNone(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Unspoken"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	sent, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if sent.Language != "" {
		t.Fatalf("language = %q, want none", sent.Language)
	}
}
