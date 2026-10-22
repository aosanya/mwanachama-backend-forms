package mwanachamaforms_test

import (
	"context"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestAnAnswerCarriesTheTargetItsMemberAnsweredThrough(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	created, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Round", OriginatorChapterID: "council", ClosesAt: futureRFC3339(24 * time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: created.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is the ward?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	for _, g := range []string{"ward-1", "ward-2"} {
		if _, err := mgr.AddTarget(ctx, mwanachamaforms.Target{FormID: created.ID, ChapterID: g}); err != nil {
			t.Fatalf("AddTarget %s: %v", g, err)
		}
	}
	if _, _, err := mgr.Publish(ctx, created.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	mgr.UseGroups(func(context.Context, string) ([]string, error) { return []string{"ward-2"}, nil })

	answers, err := mgr.SubmitAnswers(ctx, created.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine", ChapterID: "ward-1"}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	if len(answers) != 1 || answers[0].ChapterID != "ward-2" {
		t.Fatalf("answers = %+v, want one carrying ward-2", answers)
	}
	sends, err := mgr.ListSubmissions(ctx, created.ID, "", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(sends) != 1 || sends[0].ChapterID != "ward-2" {
		t.Fatalf("sends = %+v, want one from ward-2", sends)
	}
}
