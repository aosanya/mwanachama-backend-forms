package mwanachamaforms_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func ptrFloat(v float64) *float64 { return &v }

func ptrInt(v int) *int { return &v }

func addQuestion(t *testing.T, um mwanachamaforms.FormManager, d mwanachamaforms.QuestionDraft) (mwanachamaforms.Question, []mwanachamaforms.QuestionOption) {
	t.Helper()
	q, opts, err := um.AddQuestion(context.Background(), d)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	return q, opts
}

func publish(t *testing.T, um mwanachamaforms.FormManager, formID string) {
	t.Helper()
	if _, _, err := um.Publish(context.Background(), formID, "", "admin", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func refusedNaming(t *testing.T, err error, questionID string) {
	t.Helper()
	if !errors.Is(err, mwanachamaforms.ErrInvalidAnswer) {
		t.Fatalf("err = %v, want ErrInvalidAnswer", err)
	}
	if !strings.Contains(err.Error(), questionID) {
		t.Fatalf("err = %v, does not name question %s", err, questionID)
	}
}

func TestANumberOutsideItsDeclaredBoundsIsRefusedNamingTheQuestion(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerNumber, Prompt: "Household size",
		MinValue: ptrFloat(0), MaxValue: ptrFloat(20),
	}})
	publish(t, um, f.ID)
	ctx := context.Background()

	for _, v := range []float64{-1, 21} {
		_, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueNumber: ptrFloat(v)}})
		refusedNaming(t, err, q.ID)
	}
	for _, v := range []float64{0, 20} {
		if _, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueNumber: ptrFloat(v)}}); err != nil {
			t.Fatalf("SubmitAnswers(%v): %v", v, err)
		}
	}
}

func TestAMinimumOfZeroIsABoundAndAnAbsentMinimumIsNot(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	bounded, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerCurrency, Prompt: "Contribution", MinValue: ptrFloat(0),
	}})
	open, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerCurrency, Prompt: "Balance",
	}})
	publish(t, um, f.ID)
	ctx := context.Background()

	stored, err := um.ListQuestions(ctx, f.ID)
	if err != nil {
		t.Fatalf("ListQuestions: %v", err)
	}
	for _, q := range stored {
		switch q.ID {
		case bounded.ID:
			if q.MinValue == nil || *q.MinValue != 0 {
				t.Fatalf("bounded min_value = %v, want 0", q.MinValue)
			}
		case open.ID:
			if q.MinValue != nil {
				t.Fatalf("open min_value = %v, want absent", *q.MinValue)
			}
		}
	}

	_, err = um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: bounded.ID, ValueNumber: ptrFloat(-5)}})
	refusedNaming(t, err, bounded.ID)
	if _, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: open.ID, ValueNumber: ptrFloat(-5)}}); err != nil {
		t.Fatalf("SubmitAnswers(open): %v", err)
	}
}

func TestADateOutsideItsDeclaredBoundsIsRefused(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerDate, Prompt: "Date of the meeting",
		MinDate: "2026-01-01", MaxDate: "2026-12-31",
	}})
	publish(t, um, f.ID)
	ctx := context.Background()

	for _, d := range []string{"2025-12-31", "2027-01-01"} {
		_, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueDate: d}})
		refusedNaming(t, err, q.ID)
	}
	for _, d := range []string{"2026-01-01", "2026-12-31"} {
		if _, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueDate: d}}); err != nil {
			t.Fatalf("SubmitAnswers(%s): %v", d, err)
		}
	}
}

func TestAMultiSelectOutsideItsDeclaredSelectionCountIsRefused(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, opts := addQuestion(t, um, mwanachamaforms.QuestionDraft{
		Question: mwanachamaforms.Question{
			FormID: f.ID, AnswerType: models.AnswerMultiSelect, Prompt: "Which services?",
			MinSelections: ptrInt(2), MaxSelections: ptrInt(3),
		},
		Options: []mwanachamaforms.QuestionOption{{Label: "Water"}, {Label: "Roads"}, {Label: "Clinics"}, {Label: "Schools"}},
	})
	publish(t, um, f.ID)
	ctx := context.Background()

	ids := func(n int) []string {
		out := []string{}
		for _, o := range opts[:n] {
			out = append(out, o.ID)
		}
		return out
	}
	for _, n := range []int{1, 4} {
		_, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, OptionIDs: ids(n)}})
		refusedNaming(t, err, q.ID)
	}
	for _, n := range []int{2, 3} {
		if _, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, OptionIDs: ids(n)}}); err != nil {
			t.Fatalf("SubmitAnswers(%d picks): %v", n, err)
		}
	}
}

func TestASubmissionLeavingARequiredQuestionUnansweredIsRefused(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	required, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerYesNo, Prompt: "Do you consent?", Required: true,
	}})
	optional, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "Anything else?",
	}})
	publish(t, um, f.ID)
	ctx := context.Background()

	_, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: optional.ID, ValueText: "no"}})
	refusedNaming(t, err, required.ID)
	answers, err := um.ListAnswers(ctx, f.ID, "member-1")
	if err != nil {
		t.Fatalf("ListAnswers: %v", err)
	}
	if len(answers) != 0 {
		t.Fatalf("a refused submission wrote %d answers", len(answers))
	}

	yes := true
	if _, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: required.ID, ValueBool: &yes}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
}

func TestOnlyTheCurrentVersionOfARequiredQuestionIsHeldToIt(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	first, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerYesNo, Prompt: "Do you consent?", Required: true,
	}})
	publish(t, um, f.ID)
	ctx := context.Background()

	next, err := um.VersionQuestion(ctx, first.ID, "admin", mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		Prompt: "Do you consent to be contacted?", Required: true,
	}})
	if err != nil {
		t.Fatalf("VersionQuestion: %v", err)
	}
	if !next.Question.Required {
		t.Fatal("the successor lost required")
	}

	yes := true
	_, err = um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: first.ID, ValueBool: &yes}})
	refusedNaming(t, err, next.Question.ID)
	if _, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: next.Question.ID, ValueBool: &yes}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
}

func TestAQuestionWhoseValidatorsCannotBeMetIsRefused(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	cases := map[string]mwanachamaforms.Question{
		"min above max":            {AnswerType: models.AnswerNumber, MinValue: ptrFloat(5), MaxValue: ptrFloat(1)},
		"value bound on free text": {AnswerType: mwanachamaforms.AnswerFreeText, MaxValue: ptrFloat(1)},
		"date bound on a number":   {AnswerType: models.AnswerNumber, MinDate: "2026-01-01"},
		"malformed date":           {AnswerType: models.AnswerDate, MinDate: "01/01/2026"},
		"min date after max date":  {AnswerType: models.AnswerDate, MinDate: "2026-02-01", MaxDate: "2026-01-01"},
		"selections on one choice": {AnswerType: mwanachamaforms.AnswerSingleChoice, MaxSelections: ptrInt(2)},
		"zero min selections":      {AnswerType: models.AnswerMultiSelect, MinSelections: ptrInt(0)},
		"min selections above max": {AnswerType: models.AnswerMultiSelect, MinSelections: ptrInt(3), MaxSelections: ptrInt(2)},
	}
	for name, q := range cases {
		q.FormID, q.Prompt = f.ID, name
		_, _, err := um.AddQuestion(context.Background(), mwanachamaforms.QuestionDraft{Question: q})
		if !errors.Is(err, mwanachamaforms.ErrInvalidQuestion) {
			t.Errorf("%s: err = %v, want ErrInvalidQuestion", name, err)
		}
	}
}

func TestUpdatingAQuestionReplacesItsValidators(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerNumber, Prompt: "Age", Required: true, MinValue: ptrFloat(18),
	}})
	got, _, err := um.UpdateQuestion(context.Background(), f.ID, q.ID, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		AnswerType: models.AnswerNumber, Prompt: "Age", MaxValue: ptrFloat(120),
	}})
	if err != nil {
		t.Fatalf("UpdateQuestion: %v", err)
	}
	if got.Required || got.MinValue != nil || got.MaxValue == nil || *got.MaxValue != 120 {
		t.Fatalf("validators after update: required=%v min=%v max=%v", got.Required, got.MinValue, got.MaxValue)
	}
	stored, err := um.ListQuestions(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListQuestions: %v", err)
	}
	if stored[0].Required || stored[0].MinValue != nil || stored[0].MaxValue == nil {
		t.Fatalf("stored validators: %+v", stored[0])
	}
}

func TestProvisionAddsTheValidatorColumnsToAQuestionTableThatPredatesThem(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s := testSpec(t, "test")
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	o, _ := s.ByRole("question")
	table := s.TableFor(o)
	for _, column := range []string{"required", "min_value", "max_value", "min_date", "max_date", "min_selections", "max_selections"} {
		if err := db.Exec("alter table " + table + " drop column " + column).Error; err != nil {
			t.Fatalf("drop %s: %v", column, err)
		}
	}
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision over the older table: %v", err)
	}
	um, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	f := newDraftForm(t, um)
	q, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerNumber, Prompt: "Count", Required: true, MinValue: ptrFloat(0),
	}})
	stored, err := um.ListQuestions(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListQuestions: %v", err)
	}
	if len(stored) != 1 || stored[0].ID != q.ID || !stored[0].Required || stored[0].MinValue == nil || *stored[0].MinValue != 0 {
		t.Fatalf("stored: %+v", stored)
	}
}
