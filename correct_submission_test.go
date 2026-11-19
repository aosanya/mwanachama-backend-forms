package mwanachamaforms_test

import (
	"context"
	"errors"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"testing"
)

func sentOnce(t *testing.T) voiceRound {
	t.Helper()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	_, err := r.m.SubmitAnswersAs(context.Background(), r.form.ID, "mem-1", "g-1", []models.Answer{{QuestionID: r.question.ID, ValueText: "bin collection"}, {QuestionID: r.reason.ID, ValueText: "missed twice"}})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	return r
}
func TestACorrectionChangesTheNamedAnswersAndKeepsTheRest(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	got, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "clerk-1", []models.Answer{{QuestionID: r.question.ID, ValueText: "street lighting"}})
	if err != nil {
		t.Fatalf("CorrectSubmission: %v", err)
	}
	if len(got) != 1 || got[0].ValueText != "street lighting" {
		t.Fatalf("corrected answers = %+v", got)
	}
	held, err := r.m.ListAnswers(ctx, r.form.ID, "mem-1")
	if err != nil || len(held) != 2 {
		t.Fatalf("answers after the correction = %+v (%v), want both", held, err)
	}
	seen := map[string]string{}
	for _, a := range held {
		seen[a.QuestionID] = a.ValueText
	}
	if seen[r.question.ID] != "street lighting" || seen[r.reason.ID] != "missed twice" {
		t.Fatalf("answers after the correction = %v", seen)
	}
	sent, err := r.m.ListSubmissions(ctx, r.form.ID, "mem-1", "", "")
	if err != nil || len(sent) != 1 || sent[0].EditedAt == "" || sent[0].Answers != 2 {
		t.Fatalf("send after the correction = %+v (%v), want one send stamped edited carrying 2 answers", sent, err)
	}
}
func TestACorrectionIsRefusedWhenNothingWasSentOrTheQuestionIsNotOnTheForm(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	fix := []models.Answer{{QuestionID: r.question.ID, ValueText: "x"}}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-9", "clerk-1", fix); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("nothing sent = %v, want ErrSubmissionNotFound", err)
	}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "clerk-1", []models.Answer{{QuestionID: "nowhere", ValueText: "x"}}); !errors.Is(err, forms.ErrQuestionNotOnForm) {
		t.Fatalf("question off the form = %v, want ErrQuestionNotOnForm", err)
	}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "", fix); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("nobody correcting = %v, want ErrInvalidAnswer", err)
	}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "clerk-1", nil); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("nothing to correct = %v, want ErrInvalidAnswer", err)
	}
}
func TestACorrectionOfAWithdrawnSendIsRefused(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "clerk-1", []models.Answer{{QuestionID: r.question.ID, ValueText: "x"}}); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("withdrawn send = %v, want ErrSubmissionNotFound", err)
	}
}
