package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func addPrompts(t *testing.T, um mwanachamaforms.FormManager, formID string, prompts ...string) []string {
	t.Helper()
	ids := make([]string, 0, len(prompts))
	for _, p := range prompts {
		q, _, err := um.AddQuestion(context.Background(), mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
			FormID: formID, AnswerType: mwanachamaforms.AnswerSingleChoice, Prompt: p,
		}, Options: []mwanachamaforms.QuestionOption{{Label: "A"}, {Label: "B"}}})
		if err != nil {
			t.Fatalf("AddQuestion %s: %v", p, err)
		}
		ids = append(ids, q.ID)
	}
	return ids
}

func promptOrder(t *testing.T, um mwanachamaforms.FormManager, formID string) []string {
	t.Helper()
	qs, err := um.ListQuestions(context.Background(), formID)
	if err != nil {
		t.Fatalf("ListQuestions: %v", err)
	}
	out := make([]string, 0, len(qs))
	for i, q := range qs {
		if q.Ordinal != i+1 {
			t.Fatalf("ordinal %d at place %d", q.Ordinal, i+1)
		}
		out = append(out, q.Prompt)
	}
	return out
}

func sameOrder(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestF89_RemovingADraftQuestionDropsItsChoicesAndClosesTheGap(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newDraftForm(t, um)
	ids := addPrompts(t, um, f.ID, "one", "two", "three")

	if err := um.RemoveQuestion(ctx, f.ID, ids[1]); err != nil {
		t.Fatalf("RemoveQuestion: %v", err)
	}
	if got := promptOrder(t, um, f.ID); !sameOrder(got, "one", "three") {
		t.Fatalf("order = %v", got)
	}
	opts, err := um.ListOptions(ctx, ids[1])
	if err != nil {
		t.Fatalf("ListOptions: %v", err)
	}
	if len(opts) != 0 {
		t.Fatalf("removed question still has %d choices", len(opts))
	}
	addPrompts(t, um, f.ID, "four")
	if got := promptOrder(t, um, f.ID); !sameOrder(got, "one", "three", "four") {
		t.Fatalf("order after add = %v", got)
	}
}

func TestF89_MovingADraftQuestionRenumbersTheOthers(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newDraftForm(t, um)
	ids := addPrompts(t, um, f.ID, "one", "two", "three")

	qs, err := um.MoveQuestion(ctx, f.ID, ids[2], 1)
	if err != nil {
		t.Fatalf("MoveQuestion: %v", err)
	}
	if len(qs) != 3 || qs[0].ID != ids[2] {
		t.Fatalf("returned %+v", qs)
	}
	if got := promptOrder(t, um, f.ID); !sameOrder(got, "three", "one", "two") {
		t.Fatalf("order = %v", got)
	}
	if _, err := um.MoveQuestion(ctx, f.ID, ids[2], 3); err != nil {
		t.Fatalf("MoveQuestion to end: %v", err)
	}
	if got := promptOrder(t, um, f.ID); !sameOrder(got, "one", "two", "three") {
		t.Fatalf("order = %v", got)
	}
	for _, bad := range []int{0, 4} {
		if _, err := um.MoveQuestion(ctx, f.ID, ids[0], bad); !errors.Is(err, mwanachamaforms.ErrInvalidQuestion) {
			t.Fatalf("position %d: err = %v, want ErrInvalidQuestion", bad, err)
		}
	}
}

func TestF89_AQuestionOnAnotherFormIsNotFound(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newDraftForm(t, um)
	other := newDraftForm(t, um)
	ids := addPrompts(t, um, f.ID, "one")
	if err := um.RemoveQuestion(ctx, other.ID, ids[0]); !errors.Is(err, mwanachamaforms.ErrQuestionNotFound) {
		t.Fatalf("remove: err = %v", err)
	}
	if _, err := um.MoveQuestion(ctx, other.ID, ids[0], 1); !errors.Is(err, mwanachamaforms.ErrQuestionNotFound) {
		t.Fatalf("move: err = %v", err)
	}
}

func TestF89_RemoveAndMoveAreRefusedOncePublished(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newDraftForm(t, um)
	ids := addPrompts(t, um, f.ID, "one", "two")
	if _, _, err := um.Publish(ctx, f.ID, "", "admin", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := um.RemoveQuestion(ctx, f.ID, ids[0]); !errors.Is(err, mwanachamaforms.ErrNotDraft) {
		t.Fatalf("remove: err = %v, want ErrNotDraft", err)
	}
	if _, err := um.MoveQuestion(ctx, f.ID, ids[1], 1); !errors.Is(err, mwanachamaforms.ErrNotDraft) {
		t.Fatalf("move: err = %v, want ErrNotDraft", err)
	}
	if got := promptOrder(t, um, f.ID); !sameOrder(got, "one", "two") {
		t.Fatalf("order = %v", got)
	}
}
