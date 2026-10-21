package mwanachamaforms_test

import (
	"context"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func twoQuestionForm(t *testing.T, mgr mwanachamaforms.FormManager) (mwanachamaforms.Form, mwanachamaforms.Question, mwanachamaforms.Question) {
	t.Helper()
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Two", OriginatorChapterID: "council", ClosesAt: futureRFC3339(24 * time.Hour)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	var qs []mwanachamaforms.Question
	for _, prompt := range []string{"First?", "Second?"} {
		q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
			FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: prompt,
		}})
		if err != nil {
			t.Fatalf("AddQuestion: %v", err)
		}
		qs = append(qs, q)
	}
	if _, err := mgr.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: "ward-1"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := mgr.Publish(ctx, f.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f, qs[0], qs[1]
}

func send(t *testing.T, mgr mwanachamaforms.FormManager, formID, member string, answers ...mwanachamaforms.Answer) {
	t.Helper()
	if _, err := mgr.SubmitAnswers(context.Background(), formID, member, answers); err != nil {
		t.Fatalf("SubmitAnswers by %s: %v", member, err)
	}
}

func TestASendRemovesTheAnswersItNoLongerCarries(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q1, q2 := twoQuestionForm(t, mgr)

	send(t, mgr, f.ID, "alice", mwanachamaforms.Answer{QuestionID: q1.ID, ValueText: "one"}, mwanachamaforms.Answer{QuestionID: q2.ID, ValueText: "two"})
	first, _ := mgr.ListAnswers(ctx, f.ID, "alice")
	if len(first) != 2 {
		t.Fatalf("first send = %+v, want two answers", first)
	}

	send(t, mgr, f.ID, "alice", mwanachamaforms.Answer{QuestionID: q1.ID, ValueText: "one again"})
	got, _ := mgr.ListAnswers(ctx, f.ID, "alice")
	if len(got) != 1 || got[0].QuestionID != q1.ID || got[0].ValueText != "one again" {
		t.Fatalf("after skipping the second = %+v, want only the first, edited", got)
	}
	if got[0].ID != first[0].ID || got[0].AnsweredAt != first[0].AnsweredAt || got[0].EditedAt == "" {
		t.Fatalf("the carried answer was not edited in place: before %+v, after %+v", first[0], got[0])
	}
	s, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || s.Answers != 1 {
		t.Fatalf("submission = %+v, %v, want one answer on record", s, err)
	}
}

func TestAnAllEmptySendKeepsTheFormAnsweredWithNoAnswers(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q1, q2 := twoQuestionForm(t, mgr)

	send(t, mgr, f.ID, "alice", mwanachamaforms.Answer{QuestionID: q1.ID, ValueText: "one"}, mwanachamaforms.Answer{QuestionID: q2.ID, ValueText: "two"})
	send(t, mgr, f.ID, "alice")

	if got, _ := mgr.ListAnswers(ctx, f.ID, "alice"); len(got) != 0 {
		t.Fatalf("after an empty send, answers = %+v, want none", got)
	}
	s, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || s.Answers != 0 || s.EditedAt == "" {
		t.Fatalf("submission = %+v, %v, want an edited send with no answers", s, err)
	}
	if rows, _ := mgr.ListAnswered(ctx, "alice"); len(rows) != 1 || rows[0].FormID != f.ID {
		t.Fatalf("answered = %+v, want the form still answered", rows)
	}
}

func TestASendRemovesOnlyTheCallersAnswersOnThatForm(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q1, q2 := twoQuestionForm(t, mgr)
	other, o1, _ := twoQuestionForm(t, mgr)

	send(t, mgr, f.ID, "alice", mwanachamaforms.Answer{QuestionID: q1.ID, ValueText: "a"}, mwanachamaforms.Answer{QuestionID: q2.ID, ValueText: "b"})
	send(t, mgr, f.ID, "bob", mwanachamaforms.Answer{QuestionID: q1.ID, ValueText: "c"}, mwanachamaforms.Answer{QuestionID: q2.ID, ValueText: "d"})
	send(t, mgr, other.ID, "alice", mwanachamaforms.Answer{QuestionID: o1.ID, ValueText: "e"})

	send(t, mgr, f.ID, "alice")

	if got, _ := mgr.ListAnswers(ctx, f.ID, "bob"); len(got) != 2 {
		t.Fatalf("bob's answers = %+v, want both kept", got)
	}
	if got, _ := mgr.ListAnswers(ctx, other.ID, "alice"); len(got) != 1 {
		t.Fatalf("alice's answers on the other form = %+v, want kept", got)
	}
}
