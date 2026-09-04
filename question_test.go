package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestAddQuestion(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, opts, err := um.AddQuestion(context.Background(), models.Question{
		FormID: f.ID, AnswerType: models.AnswerSingleChoice, Prompt: "Favorite color?",
	}, []models.QuestionOption{{Label: "Red"}, {Label: "Blue"}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if q.Ordinal != 1 || q.Version != 1 || len(opts) != 2 || opts[0].Ordinal != 1 {
		t.Fatalf("unexpected question: %+v %+v", q, opts)
	}

	q2, _, err := um.AddQuestion(context.Background(), models.Question{
		FormID: f.ID, AnswerType: models.AnswerFreeText, Prompt: "Anything else?",
	}, nil)
	if err != nil {
		t.Fatalf("AddQuestion 2: %v", err)
	}
	if q2.Ordinal != 2 {
		t.Fatalf("unexpected ordinal: %d", q2.Ordinal)
	}
}

func TestAddQuestion_NotDraft(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	_, _, err := um.AddQuestion(context.Background(), models.Question{FormID: f.ID, AnswerType: models.AnswerFreeText, Prompt: "x"}, nil)
	if !errors.Is(err, mwanachamaforms.ErrNotDraft) {
		t.Fatalf("err = %v, want ErrNotDraft", err)
	}
}

func TestUpdateQuestion(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _, err := um.AddQuestion(context.Background(), models.Question{
		FormID: f.ID, AnswerType: models.AnswerSingleChoice, Prompt: "Original",
	}, []models.QuestionOption{{Label: "A"}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}

	out, opts, err := um.UpdateQuestion(context.Background(), f.ID, q.ID, models.Question{
		AnswerType: models.AnswerSingleChoice, Prompt: "Updated",
	}, []models.QuestionOption{{Label: "X"}, {Label: "Y"}})
	if err != nil {
		t.Fatalf("UpdateQuestion: %v", err)
	}
	if out.Prompt != "Updated" || out.Ordinal != q.Ordinal || out.ID != q.ID || len(opts) != 2 {
		t.Fatalf("unexpected question: %+v %+v", out, opts)
	}
}

func TestUpdateQuestion_MissingPrompt(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _, err := um.AddQuestion(context.Background(), models.Question{FormID: f.ID, AnswerType: models.AnswerFreeText, Prompt: "x"}, nil)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	_, _, err = um.UpdateQuestion(context.Background(), f.ID, q.ID, models.Question{Prompt: "  "}, nil)
	if !errors.Is(err, mwanachamaforms.ErrMissingPrompt) {
		t.Fatalf("err = %v, want ErrMissingPrompt", err)
	}
}

func TestAddOption_AllowedAfterPublish(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _, err := um.AddQuestion(context.Background(), models.Question{
		FormID: f.ID, AnswerType: models.AnswerSingleChoice, Prompt: "Pick one",
	}, []models.QuestionOption{{Label: "A"}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	result, err := um.AddOption(context.Background(), q.ID, "B", "editor")
	if err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	if result.OptionsBefore != 1 || result.OptionsAfter != 2 || result.Option.Ordinal != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVersionQuestion(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, opts, err := um.AddQuestion(context.Background(), models.Question{
		FormID: f.ID, AnswerType: models.AnswerSingleChoice, Prompt: "Original wording",
	}, []models.QuestionOption{{Label: "A"}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	result, err := um.VersionQuestion(context.Background(), q.ID, models.Question{Prompt: "Clearer wording"}, []models.QuestionOption{{Label: "A"}, {Label: "B"}}, "editor")
	if err != nil {
		t.Fatalf("VersionQuestion: %v", err)
	}
	if result.Question.ID == q.ID || result.Question.SupersedesQuestionID != q.ID ||
		result.Question.Version != 2 || result.Question.Ordinal != q.Ordinal || len(result.Options) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}

	// The predecessor's own options are untouched.
	predOpts, err := um.ListOptions(context.Background(), q.ID)
	if err != nil {
		t.Fatalf("ListOptions(predecessor): %v", err)
	}
	if len(predOpts) != len(opts) {
		t.Fatalf("predecessor options changed: %+v", predOpts)
	}

	// Versioning the predecessor again is refused — it is no longer current.
	if _, err := um.VersionQuestion(context.Background(), q.ID, models.Question{Prompt: "Again"}, nil, "editor"); !errors.Is(err, mwanachamaforms.ErrNotCurrentVersion) {
		t.Fatalf("err = %v, want ErrNotCurrentVersion", err)
	}
}

func TestVersionQuestion_DraftFormRefused(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _, err := um.AddQuestion(context.Background(), models.Question{FormID: f.ID, AnswerType: models.AnswerFreeText, Prompt: "x"}, nil)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := um.VersionQuestion(context.Background(), q.ID, models.Question{Prompt: "y"}, nil, "editor"); !errors.Is(err, mwanachamaforms.ErrFormNotPublished) {
		t.Fatalf("err = %v, want ErrFormNotPublished", err)
	}
}
