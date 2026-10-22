package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func publishedToTwoGroups(t *testing.T) (mwanachamaforms.FormManager, string, string) {
	t.Helper()
	mgr := newTestManager(t)
	ctx := context.Background()
	created, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Round", OriginatorChapterID: "council", ClosesAt: futureRFC3339(24 * time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: created.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is trade?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	for _, g := range []string{"traders", "teachers", "farmers"} {
		if _, err := mgr.AddTarget(ctx, mwanachamaforms.Target{FormID: created.ID, ChapterID: g}); err != nil {
			t.Fatalf("AddTarget %s: %v", g, err)
		}
	}
	if _, _, err := mgr.Publish(ctx, created.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	mgr.UseGroups(func(context.Context, string) ([]string, error) { return []string{"traders", "teachers"}, nil })
	return mgr, created.ID, q.ID
}

func TestAMemberAnswersOncePerGroupTheyAnswerAs(t *testing.T) {
	mgr, formID, qID := publishedToTwoGroups(t)
	ctx := context.Background()
	for _, g := range []string{"traders", "teachers"} {
		if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", g, []mwanachamaforms.Answer{{QuestionID: qID, ValueText: "as " + g}}); err != nil {
			t.Fatalf("SubmitAnswersAs %s: %v", g, err)
		}
	}
	if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", "traders", []mwanachamaforms.Answer{{QuestionID: qID, ValueText: "edited"}}); err != nil {
		t.Fatalf("resend as traders: %v", err)
	}
	sends, err := mgr.ListSubmissions(ctx, formID, "alice")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(sends) != 2 || sends[0].ChapterID != "teachers" && sends[1].ChapterID != "teachers" {
		t.Fatalf("sends = %+v, want one each for traders and teachers", sends)
	}
	for _, s := range sends {
		if s.Answers != 1 {
			t.Fatalf("send %+v carries %d answers, want 1", s, s.Answers)
		}
	}
	answers, err := mgr.ListAnswers(ctx, formID, "alice")
	if err != nil {
		t.Fatalf("ListAnswers: %v", err)
	}
	byGroup := map[string]string{}
	for _, a := range answers {
		byGroup[a.ChapterID] = a.ValueText
	}
	if byGroup["traders"] != "edited" || byGroup["teachers"] != "as teachers" {
		t.Fatalf("answers by group = %v, want the traders answer edited and the teachers one kept", byGroup)
	}
}

func TestAMemberCannotAnswerAsAGroupTheyAreNotIn(t *testing.T) {
	mgr, formID, qID := publishedToTwoGroups(t)
	ctx := context.Background()
	for _, g := range []string{"farmers", "nobody"} {
		_, err := mgr.SubmitAnswersAs(ctx, formID, "alice", g, []mwanachamaforms.Answer{{QuestionID: qID, ValueText: "x"}})
		if !errors.Is(err, mwanachamaforms.ErrInvalidAnswer) {
			t.Fatalf("answering as %s: err = %v, want ErrInvalidAnswer", g, err)
		}
	}
}

func TestWaitingAndAnsweredReportEachGroupAnsweredAs(t *testing.T) {
	mgr, formID, qID := publishedToTwoGroups(t)
	ctx := context.Background()
	if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", "teachers", []mwanachamaforms.Answer{{QuestionID: qID, ValueText: "ok"}}); err != nil {
		t.Fatalf("SubmitAnswersAs: %v", err)
	}
	waiting, err := mgr.ListWaiting(ctx, "alice")
	if err != nil {
		t.Fatalf("ListWaiting: %v", err)
	}
	if len(waiting) != 1 || len(waiting[0].AnsweredAs) != 1 || waiting[0].AnsweredAs[0] != "teachers" {
		t.Fatalf("waiting = %+v, want answered as teachers only", waiting)
	}
	if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", "traders", []mwanachamaforms.Answer{{QuestionID: qID, ValueText: "ok"}}); err != nil {
		t.Fatalf("SubmitAnswersAs: %v", err)
	}
	answered, err := mgr.ListAnswered(ctx, "alice")
	if err != nil {
		t.Fatalf("ListAnswered: %v", err)
	}
	if len(answered) != 2 || answered[0].ChapterID != "teachers" || answered[1].ChapterID != "traders" {
		t.Fatalf("answered = %+v, want one row per group, teachers then traders", answered)
	}
}

func TestAFormStaysWaitingWhileAGroupTheMemberMayAnswerAsIsUnanswered(t *testing.T) {
	mgr, formID, qID := publishedToTwoGroups(t)
	ctx := context.Background()
	waitingFor := func() (bool, []string) {
		t.Helper()
		waiting, err := mgr.ListWaiting(ctx, "alice")
		if err != nil || len(waiting) != 1 {
			t.Fatalf("ListWaiting = %+v, %v; want the one form", waiting, err)
		}
		return waiting[0].Answered, waiting[0].UnansweredAs
	}
	if answered, left := waitingFor(); answered || len(left) != 2 || left[0] != "teachers" || left[1] != "traders" {
		t.Fatalf("before any send: answered=%v left=%v, want unanswered as teachers and traders", answered, left)
	}
	if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", "teachers", []mwanachamaforms.Answer{{QuestionID: qID, ValueText: "ok"}}); err != nil {
		t.Fatalf("SubmitAnswersAs teachers: %v", err)
	}
	if answered, left := waitingFor(); answered || len(left) != 1 || left[0] != "traders" {
		t.Fatalf("after teachers: answered=%v left=%v, want still waiting as traders", answered, left)
	}
	if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", "traders", nil); err != nil {
		t.Fatalf("SubmitAnswersAs traders: %v", err)
	}
	if answered, left := waitingFor(); !answered || len(left) != 0 {
		t.Fatalf("after both: answered=%v left=%v, want answered with nothing left", answered, left)
	}
}
