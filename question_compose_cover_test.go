package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func draftWithQuestions(t *testing.T, m forms.FormManager, prompts ...string) (models.Form, []models.Question) {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic compose round", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	var qs []models.Question
	for _, p := range prompts {
		q, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: p, AnswerType: models.AnswerFreeText}})
		if err != nil {
			t.Fatalf("AddQuestion %q: %v", p, err)
		}
		qs = append(qs, q)
	}
	return f, qs
}

func promptsOf(qs []models.Question) []string {
	out := []string{}
	for _, q := range qs {
		out = append(out, q.Prompt)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMovingAQuestionRenumbersTheWholeForm(t *testing.T) {
	m := gatedManager(t, nil)
	f, qs := draftWithQuestions(t, m, "A", "B", "C", "D")

	got, err := m.MoveQuestion(context.Background(), f.ID, qs[3].ID, 1)
	if err != nil {
		t.Fatalf("MoveQuestion: %v", err)
	}
	if want := []string{"D", "A", "B", "C"}; !equalStrings(promptsOf(got), want) {
		t.Fatalf("order after move to first = %v, want %v", promptsOf(got), want)
	}
	for i, q := range got {
		if q.Ordinal != i+1 {
			t.Fatalf("question %q ordinal = %d, want %d", q.Prompt, q.Ordinal, i+1)
		}
	}

	got, err = m.MoveQuestion(context.Background(), f.ID, qs[3].ID, 4)
	if err != nil {
		t.Fatalf("MoveQuestion: %v", err)
	}
	if want := []string{"A", "B", "C", "D"}; !equalStrings(promptsOf(got), want) {
		t.Fatalf("order after move to last = %v, want %v", promptsOf(got), want)
	}
}

func TestMovingAQuestionOutsideTheFormIsRefusedAndLeavesTheOrder(t *testing.T) {
	m := gatedManager(t, nil)
	f, qs := draftWithQuestions(t, m, "A", "B", "C")

	for _, position := range []int{0, -1, 4} {
		_, err := m.MoveQuestion(context.Background(), f.ID, qs[0].ID, position)
		if !errors.Is(err, forms.ErrInvalidQuestion) {
			t.Fatalf("MoveQuestion to %d = %v, want ErrInvalidQuestion", position, err)
		}
	}
	after, _ := m.ListQuestions(context.Background(), f.ID)
	if want := []string{"A", "B", "C"}; !equalStrings(promptsOf(after), want) {
		t.Fatalf("order after refused moves = %v, want %v", promptsOf(after), want)
	}
}

func TestRemovingAQuestionClosesTheGapAndDropsItsOptions(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	f, qs := draftWithQuestions(t, m, "A", "C")
	choice, opts, err := m.AddQuestion(ctx, models.QuestionDraft{
		Question: models.Question{FormID: f.ID, Prompt: "B", AnswerType: models.AnswerSingleChoice},
		Options:  []models.QuestionOption{{Label: "yes"}, {Label: "no"}},
	})
	if err != nil || len(opts) != 2 {
		t.Fatalf("AddQuestion choice: %v, %d options", err, len(opts))
	}

	if err := m.RemoveQuestion(ctx, f.ID, qs[0].ID); err != nil {
		t.Fatalf("RemoveQuestion: %v", err)
	}
	left, _ := m.ListQuestions(ctx, f.ID)
	if want := []string{"C", "B"}; !equalStrings(promptsOf(left), want) {
		t.Fatalf("prompts after removing first = %v, want %v", promptsOf(left), want)
	}
	for i, q := range left {
		if q.Ordinal != i+1 {
			t.Fatalf("%q ordinal = %d, want %d", q.Prompt, q.Ordinal, i+1)
		}
	}

	if err := m.RemoveQuestion(ctx, f.ID, choice.ID); err != nil {
		t.Fatalf("RemoveQuestion choice: %v", err)
	}
	gone, _ := m.ListOptions(ctx, choice.ID)
	if len(gone) != 0 {
		t.Fatalf("%d options survived their question's removal", len(gone))
	}
}

func TestComposingRefusesAQuestionOfAnotherFormAndAPublishedForm(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	f, qs := draftWithQuestions(t, m, "A")
	other, _ := draftWithQuestions(t, m, "Z")

	if err := m.RemoveQuestion(ctx, other.ID, qs[0].ID); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("RemoveQuestion across forms = %v, want ErrQuestionNotFound", err)
	}
	if _, err := m.MoveQuestion(ctx, other.ID, qs[0].ID, 1); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("MoveQuestion across forms = %v, want ErrQuestionNotFound", err)
	}
	if err := m.RemoveQuestion(ctx, f.ID, "no-such-question"); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("RemoveQuestion unknown = %v, want ErrQuestionNotFound", err)
	}
	if err := m.RemoveQuestion(ctx, "no-such-form", qs[0].ID); err == nil {
		t.Fatal("RemoveQuestion on an unknown form succeeded")
	}

	target(t, m, f, "g-1", false)
	publishForm(t, m, f)
	if err := m.RemoveQuestion(ctx, f.ID, qs[0].ID); !errors.Is(err, forms.ErrNotDraft) {
		t.Fatalf("RemoveQuestion on published = %v, want ErrNotDraft", err)
	}
	if _, err := m.MoveQuestion(ctx, f.ID, qs[0].ID, 1); !errors.Is(err, forms.ErrNotDraft) {
		t.Fatalf("MoveQuestion on published = %v, want ErrNotDraft", err)
	}
	if _, _, err := m.UpdateQuestion(ctx, f.ID, qs[0].ID, models.QuestionDraft{Question: models.Question{Prompt: "New", AnswerType: models.AnswerFreeText}}); !errors.Is(err, forms.ErrNotDraft) {
		t.Fatalf("UpdateQuestion on published = %v, want ErrNotDraft", err)
	}
}

func TestUpdatingAQuestionReplacesItsPromptAndOptions(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	f, _ := draftWithQuestions(t, m)
	q, _, err := m.AddQuestion(ctx, models.QuestionDraft{
		Question: models.Question{FormID: f.ID, Prompt: "Pick", AnswerType: models.AnswerSingleChoice},
		Options:  []models.QuestionOption{{Label: "one"}, {Label: "two"}, {Label: "three"}},
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}

	updated, opts, err := m.UpdateQuestion(ctx, f.ID, q.ID, models.QuestionDraft{
		Question: models.Question{Prompt: "Pick again", AnswerType: models.AnswerMultiSelect, Required: true},
		Options:  []models.QuestionOption{{Label: "x"}, {Label: "y"}},
	})
	if err != nil {
		t.Fatalf("UpdateQuestion: %v", err)
	}
	if updated.Prompt != "Pick again" || updated.AnswerType != models.AnswerMultiSelect || !updated.Required {
		t.Fatalf("updated = %+v", updated)
	}
	if updated.FormID != f.ID || updated.ID != q.ID {
		t.Fatalf("update changed identity: %+v", updated)
	}
	if len(opts) != 2 || opts[0].Label != "x" || opts[0].Ordinal != 1 || opts[1].Label != "y" || opts[1].Ordinal != 2 {
		t.Fatalf("returned options = %+v", opts)
	}
	stored, _ := m.ListOptions(ctx, q.ID)
	if len(stored) != 2 || stored[0].Label != "x" || stored[1].Label != "y" {
		t.Fatalf("stored options = %+v, want x then y only", stored)
	}
	qs, _ := m.ListQuestions(ctx, f.ID)
	if len(qs) != 1 || qs[0].Prompt != "Pick again" {
		t.Fatalf("stored questions = %+v", qs)
	}
}

func TestUpdatingAQuestionRefusesBlankPromptBlankOptionAndForeignQuestion(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	f, qs := draftWithQuestions(t, m, "Keep")
	other, _ := draftWithQuestions(t, m, "Other")

	if _, _, err := m.UpdateQuestion(ctx, f.ID, qs[0].ID, models.QuestionDraft{Question: models.Question{Prompt: "   ", AnswerType: models.AnswerFreeText}}); !errors.Is(err, forms.ErrMissingPrompt) {
		t.Fatalf("blank prompt = %v, want ErrMissingPrompt", err)
	}
	blankOption := models.QuestionDraft{
		Question: models.Question{Prompt: "Keep", AnswerType: models.AnswerSingleChoice},
		Options:  []models.QuestionOption{{Label: "ok"}, {Label: " "}},
	}
	if _, _, err := m.UpdateQuestion(ctx, f.ID, qs[0].ID, blankOption); !errors.Is(err, forms.ErrMissingOptionLabel) {
		t.Fatalf("blank option = %v, want ErrMissingOptionLabel", err)
	}
	if _, _, err := m.UpdateQuestion(ctx, other.ID, qs[0].ID, models.QuestionDraft{Question: models.Question{Prompt: "Hijack", AnswerType: models.AnswerFreeText}}); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("foreign question = %v, want ErrQuestionNotFound", err)
	}
	if _, _, err := m.UpdateQuestion(ctx, f.ID, "no-such-question", models.QuestionDraft{Question: models.Question{Prompt: "X", AnswerType: models.AnswerFreeText}}); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("unknown question = %v, want ErrQuestionNotFound", err)
	}
	after, _ := m.ListQuestions(ctx, f.ID)
	if len(after) != 1 || after[0].Prompt != "Keep" {
		t.Fatalf("refused updates changed the question: %+v", after)
	}
}

func TestAddingAnOptionAppendsAndCountsBeforeAndAfter(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	f, _ := draftWithQuestions(t, m)
	q, _, err := m.AddQuestion(ctx, models.QuestionDraft{
		Question: models.Question{FormID: f.ID, Prompt: "Pick", AnswerType: models.AnswerSingleChoice},
		Options:  []models.QuestionOption{{Label: "one"}, {Label: "two"}},
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}

	res, err := m.AddOption(ctx, q.ID, "three", "someone")
	if err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	if res.OptionsBefore != 2 || res.OptionsAfter != 3 || res.Option.Ordinal != 3 || res.Option.Label != "three" || res.Option.QuestionID != q.ID {
		t.Fatalf("AddOptionResult = %+v", res)
	}
	listed, _ := m.ListOptions(ctx, q.ID)
	if len(listed) != 3 || listed[2].ID != res.Option.ID {
		t.Fatalf("listed options = %+v", listed)
	}
}

func TestAddingAnOptionRefusesBlankLabelAndUnknownQuestion(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	_, qs := draftWithQuestions(t, m, "A")

	if _, err := m.AddOption(ctx, qs[0].ID, "  ", "someone"); !errors.Is(err, forms.ErrMissingOptionLabel) {
		t.Fatalf("blank label = %v, want ErrMissingOptionLabel", err)
	}
	if _, err := m.AddOption(ctx, "", "label", "someone"); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("empty question id = %v, want ErrQuestionNotFound", err)
	}
	if _, err := m.AddOption(ctx, "no-such-question", "label", "someone"); !errors.Is(err, forms.ErrQuestionNotFound) {
		t.Fatalf("unknown question = %v, want ErrQuestionNotFound", err)
	}
	opts, _ := m.ListOptions(ctx, qs[0].ID)
	if len(opts) != 0 {
		t.Fatalf("refused AddOption left %d options", len(opts))
	}
}
