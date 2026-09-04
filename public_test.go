package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func newOpenPublicForm(t *testing.T, um mwanachamaforms.FormManager, key string) models.Form {
	t.Helper()
	f, err := um.Create(context.Background(), models.Form{
		Title: "Census", OriginatorChapterID: "chapter-1", ClosesAt: futureRFC3339(time.Hour), Audience: models.AudiencePublic,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, key, "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f
}

func TestResolveLinkKey(t *testing.T) {
	um := newTestManager(t)
	f := newOpenPublicForm(t, um, "the-key")

	link, out, found, err := um.ResolveLinkKey(context.Background(), "the-key")
	if err != nil {
		t.Fatalf("ResolveLinkKey: %v", err)
	}
	if !found || link.Key != "the-key" || out.ID != f.ID {
		t.Fatalf("unexpected result: found=%v link=%+v form=%+v", found, link, out)
	}
}

func TestResolveLinkKey_UnknownKey(t *testing.T) {
	um := newTestManager(t)
	_, _, found, err := um.ResolveLinkKey(context.Background(), "nope")
	if err != nil {
		t.Fatalf("ResolveLinkKey: %v", err)
	}
	if found {
		t.Fatal("expected found=false for an unknown key")
	}
}

func TestUpsertRespondent(t *testing.T) {
	um := newTestManager(t)
	first, err := um.UpsertRespondent(context.Background(), models.Respondent{PublicKey: "pk-1"})
	if err != nil {
		t.Fatalf("UpsertRespondent: %v", err)
	}
	if first.ID == "" || first.FirstSeenAt == "" {
		t.Fatalf("unexpected respondent: %+v", first)
	}
	second, err := um.UpsertRespondent(context.Background(), models.Respondent{PublicKey: "pk-1"})
	if err != nil {
		t.Fatalf("UpsertRespondent (repeat): %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected the same respondent back, got %+v vs %+v", second, first)
	}
}

func TestSubmitAnswers(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	yn, _, err := um.AddQuestion(context.Background(), models.Question{FormID: f.ID, AnswerType: models.AnswerYesNo, Prompt: "Active chapter?"}, nil)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	yes := true
	out, err := um.SubmitAnswers(context.Background(), f.ID, "member-1", "chapter-2", []models.Answer{
		{QuestionID: yn.ID, ValueBool: &yes},
	})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	if len(out) != 1 || out[0].EditedAt != "" || out[0].ChapterID != "chapter-2" {
		t.Fatalf("unexpected answers: %+v", out)
	}
	firstAnsweredAt := out[0].AnsweredAt

	no := false
	out2, err := um.SubmitAnswers(context.Background(), f.ID, "member-1", "chapter-3", []models.Answer{
		{QuestionID: yn.ID, ValueBool: &no},
	})
	if err != nil {
		t.Fatalf("SubmitAnswers (resubmit): %v", err)
	}
	if out2[0].EditedAt == "" || out2[0].AnsweredAt != firstAnsweredAt || out2[0].ChapterID != "chapter-2" {
		t.Fatalf("resubmit did not preserve capture time/chapter: %+v", out2)
	}
	if *out2[0].ValueBool != false {
		t.Fatalf("resubmit did not update the value: %+v", out2[0])
	}
}

func TestSubmitAnswers_FormNotOpen(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _, err := um.AddQuestion(context.Background(), models.Question{FormID: f.ID, AnswerType: models.AnswerFreeText, Prompt: "x"}, nil)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	_, err = um.SubmitAnswers(context.Background(), f.ID, "member-1", "chapter-2", []models.Answer{{QuestionID: q.ID, ValueText: "hi"}})
	if !errors.Is(err, mwanachamaforms.ErrFormNotOpen) {
		t.Fatalf("err = %v, want ErrFormNotOpen", err)
	}
}

func TestSubmitAnswers_InvalidShape(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	q, _, err := um.AddQuestion(context.Background(), models.Question{FormID: f.ID, AnswerType: models.AnswerYesNo, Prompt: "x"}, nil)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	_, err = um.SubmitAnswers(context.Background(), f.ID, "member-1", "chapter-2", []models.Answer{{QuestionID: q.ID}})
	if !errors.Is(err, mwanachamaforms.ErrInvalidAnswer) {
		t.Fatalf("err = %v, want ErrInvalidAnswer", err)
	}
}

func TestDeclare(t *testing.T) {
	um := newTestManager(t)
	f := newOpenPublicForm(t, um, "declare-key")
	r, err := um.UpsertRespondent(context.Background(), models.Respondent{PublicKey: "pk-2"})
	if err != nil {
		t.Fatalf("UpsertRespondent: %v", err)
	}
	d, err := um.Declare(context.Background(), models.Declaration{RespondentID: r.ID, FormID: f.ID, DeclaredChapterID: "chapter-9"})
	if err != nil {
		t.Fatalf("Declare: %v", err)
	}
	if d.ID == "" || d.DeclaredChapterID != "chapter-9" {
		t.Fatalf("unexpected declaration: %+v", d)
	}

	updated, err := um.Declare(context.Background(), models.Declaration{RespondentID: r.ID, FormID: f.ID, DeclaredChapterID: "chapter-10"})
	if err != nil {
		t.Fatalf("Declare (repeat): %v", err)
	}
	if updated.ID != d.ID || updated.DeclaredChapterID != "chapter-10" {
		t.Fatalf("repeat declare did not overwrite in place: %+v", updated)
	}
}
