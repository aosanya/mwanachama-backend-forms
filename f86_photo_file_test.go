package mwanachamaforms_test

import (
	"context"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func photoQuestion(maxFiles *int, capture bool) models.Question {
	return models.Question{ID: "q1", AnswerType: models.AnswerPhoto, MaxFiles: maxFiles, RequireCapture: capture}
}

func TestAPhotoAnswerHoldsFileIDs(t *testing.T) {
	q := photoQuestion(nil, false)
	if err := models.ValidateAnswer(q, nil, models.Answer{FileIDs: []string{"f1", "f2"}}); err != nil {
		t.Fatalf("two photos refused: %v", err)
	}
	if err := models.ValidateAnswer(q, nil, models.Answer{}); err == nil {
		t.Fatal("a photo answer with no file was accepted")
	}
	if err := models.ValidateAnswer(q, nil, models.Answer{FileIDs: []string{"f1", "f1"}}); err == nil {
		t.Fatal("the same file twice was accepted")
	}
	fq := models.Question{AnswerType: models.AnswerFile}
	if err := models.ValidateAnswer(fq, nil, models.Answer{FileIDs: []string{"doc"}}); err != nil {
		t.Fatalf("a file answer refused: %v", err)
	}
}

func TestTheQuestionsLimitOutranksTheHostDefault(t *testing.T) {
	six := models.Answer{FileIDs: []string{"a", "b", "c", "d", "e", "f"}}
	if err := models.ValidateAnswerWithin(photoQuestion(nil, false), nil, six, 5); err == nil {
		t.Fatal("six photos passed the host default of five")
	}
	if err := models.ValidateAnswerWithin(photoQuestion(ptrInt(6), false), nil, six, 5); err != nil {
		t.Fatalf("six photos refused under a question limit of six: %v", err)
	}
	if err := models.ValidateAnswerWithin(photoQuestion(ptrInt(2), false), nil, models.Answer{FileIDs: []string{"a", "b", "c"}}, 5); err == nil {
		t.Fatal("three photos passed a question limit of two")
	}
	if err := models.ValidateAnswerWithin(photoQuestion(nil, false), nil, six, 0); err != nil {
		t.Fatalf("no limit set anywhere still refused: %v", err)
	}
}

func TestTakenNowRefusesAPhotoWithoutTimeAndPlace(t *testing.T) {
	q := photoQuestion(nil, true)
	err := models.ValidateAnswer(q, nil, models.Answer{FileIDs: []string{"a", "b"}, CapturedFileIDs: []string{"a"}})
	if err == nil || !strings.Contains(err.Error(), `"b"`) {
		t.Fatalf("an uncaptured photo was not refused by name: %v", err)
	}
	if err := models.ValidateAnswer(q, nil, models.Answer{FileIDs: []string{"a", "b"}, CapturedFileIDs: []string{"a", "b"}}); err != nil {
		t.Fatalf("captured photos refused: %v", err)
	}
	if err := models.ValidateAnswer(photoQuestion(nil, false), nil, models.Answer{FileIDs: []string{"a"}}); err != nil {
		t.Fatalf("an uncaptured photo refused where capture is not asked: %v", err)
	}
}

func TestFileRulesApplyOnlyToFileQuestions(t *testing.T) {
	cases := []models.Question{
		{AnswerType: models.AnswerFreeText, MaxFiles: ptrInt(3)},
		{AnswerType: models.AnswerFile, RequireCapture: true},
		{AnswerType: models.AnswerPhoto, MaxFiles: ptrInt(0)},
	}
	for _, q := range cases {
		if err := models.ValidateValidators(q); err == nil {
			t.Fatalf("%+v accepted", q)
		}
	}
	if err := models.ValidateValidators(photoQuestion(ptrInt(3), true)); err != nil {
		t.Fatalf("a valid photo question refused: %v", err)
	}
}

func TestAPhotoAnswerIsStoredAndHeldToTheHostDefault(t *testing.T) {
	um := newTestManager(t)
	um.UseDefaultMaxFiles(2)
	f := newDraftForm(t, um)
	q, _ := addQuestion(t, um, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerPhoto, Prompt: "The broken light", RequireCapture: true,
	}})
	publish(t, um, f.ID)
	ctx := context.Background()

	_, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, FileIDs: []string{"a", "b", "c"}, CapturedFileIDs: []string{"a", "b", "c"}}})
	refusedNaming(t, err, q.ID)
	_, err = um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, FileIDs: []string{"a"}}})
	refusedNaming(t, err, q.ID)

	out, err := um.SubmitAnswers(ctx, f.ID, "member-1", []mwanachamaforms.Answer{{QuestionID: q.ID, FileIDs: []string{"a", "b"}, CapturedFileIDs: []string{"a", "b"}}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	if len(out) != 1 || len(out[0].FileIDs) != 2 || out[0].FileIDs[1] != "b" {
		t.Fatalf("stored answer = %+v", out)
	}
}

func TestFileIDsAreRefusedOnAQuestionThatHoldsNoFiles(t *testing.T) {
	yes := true
	n := 3.0
	cases := []struct {
		q models.Question
		a models.Answer
	}{
		{models.Question{AnswerType: models.AnswerYesNo}, models.Answer{ValueBool: &yes}},
		{models.Question{AnswerType: models.AnswerFreeText}, models.Answer{ValueText: "hello"}},
		{models.Question{AnswerType: models.AnswerNumber}, models.Answer{ValueNumber: &n}},
		{models.Question{AnswerType: models.AnswerDate}, models.Answer{ValueDate: "2026-10-22"}},
	}
	for _, c := range cases {
		if err := models.ValidateAnswer(c.q, nil, c.a); err != nil {
			t.Fatalf("%s answer without files refused: %v", c.q.AnswerType, err)
		}
		c.a.FileIDs = []string{"f1"}
		err := models.ValidateAnswer(c.q, nil, c.a)
		if err == nil || !strings.Contains(err.Error(), "file_ids") {
			t.Fatalf("%s answer carrying file_ids was not refused for them: %v", c.q.AnswerType, err)
		}
	}
}
