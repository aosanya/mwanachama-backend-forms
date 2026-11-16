package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestAWithdrawnMemberCanAnswerALockedFormAgainButOnlyOnce(t *testing.T) {
	ctx := context.Background()
	m := gatedManager(t, nil)
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic locked round", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
		LockedAfterSend: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Your view", AnswerType: models.AnswerFreeText}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := m.AddTarget(ctx, models.Target{FormID: f.ID, ChapterID: "g-1"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := m.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	send := func(view string) error {
		_, err := m.SubmitAnswersAs(ctx, f.ID, "mem-1", "g-1", []models.Answer{{QuestionID: q.ID, ValueText: view}})
		return err
	}
	if err := send("keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := send("close the market"); !errors.Is(err, forms.ErrLockedAfterSend) {
		t.Fatalf("second send = %v, want ErrLockedAfterSend", err)
	}
	if err := m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if err := send("close the market"); err != nil {
		t.Fatalf("send after withdrawal = %v, want it accepted", err)
	}
	if err := send("close it on Sundays"); !errors.Is(err, forms.ErrLockedAfterSend) {
		t.Fatalf("send after the re-answer = %v, want ErrLockedAfterSend", err)
	}
}
