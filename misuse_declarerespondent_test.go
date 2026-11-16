package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type twoLinks struct {
	ctx     context.Context
	m       forms.FormManager
	respond models.RespondentPublicView
}

func twoPublicLinks(t *testing.T) twoLinks {
	t.Helper()
	ctx := context.Background()
	m := gatedManager(t, nil)
	for _, key := range []string{"link-a", "link-b"} {
		f, err := m.Create(ctx, models.Form{
			Title: "Street survey " + key, ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-agency",
			CreatedBy: "author-1", Audience: models.AudiencePublic, CollectionMode: models.CollectionSelf,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Which street", AnswerType: models.AnswerFreeText}}); err != nil {
			t.Fatalf("AddQuestion: %v", err)
		}
		if _, _, err := m.Publish(ctx, f.ID, key, "author-1", "Town hall"); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}
	r, err := m.RegisterRespondent(ctx, "link-a", "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	return twoLinks{ctx: ctx, m: m, respond: r}
}

func TestPublicDoor_DeviceHoldingTheKeyDeclaresOnItsOwnLink(t *testing.T) {
	w := twoPublicLinks(t)
	if _, err := w.m.DeclareChapter(w.ctx, "link-a", w.respond.ID, w.respond.PublicKey, "", "Mill Street"); err != nil {
		t.Fatalf("declaration on own link refused: %v", err)
	}
}

func TestPublicDoor_RespondentOfOneLinkCannotBeDeclaredOnAnotherFormsLink(t *testing.T) {
	w := twoPublicLinks(t)
	if _, err := w.m.DeclareChapter(w.ctx, "link-b", w.respond.ID, w.respond.PublicKey, "", "Orchard Lane"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("respondent registered on link-a admitted on link-b: %v", err)
	}
}

func TestPublicDoor_RespondentIdAloneCannotDeclareForAnotherDevice(t *testing.T) {
	w := twoPublicLinks(t)
	if _, err := w.m.DeclareChapter(w.ctx, "link-a", w.respond.ID, "", "", "Mill Street"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("declaration with no key: %v", err)
	}
	if _, err := w.m.DeclareChapter(w.ctx, "link-a", w.respond.ID, "guessed-key", "", "overwritten"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("declaration with a wrong key: %v", err)
	}
	if _, err := w.m.DeclareChapter(w.ctx, "link-a", "made-up", w.respond.PublicKey, "", "x"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("declaration for a made-up respondent: %v", err)
	}
}

func TestPublicDoor_AnotherRespondentsKeyDoesNotOpenThisOne(t *testing.T) {
	w := twoPublicLinks(t)
	other, err := w.m.RegisterRespondent(w.ctx, "link-a", "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	if _, err := w.m.DeclareChapter(w.ctx, "link-a", w.respond.ID, other.PublicKey, "", "overwritten"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("a second device's key declared for the first respondent: %v", err)
	}
}

func TestPublicDoor_TokenRegisteredOnASecondLinkDeclaresThereAndKeepsEachSaying(t *testing.T) {
	w := twoPublicLinks(t)
	again, err := w.m.RegisterRespondent(w.ctx, "link-b", w.respond.PublicKey)
	if err != nil || again.ID != w.respond.ID {
		t.Fatalf("reclaiming the token on link-b = %+v, %v", again, err)
	}
	a, err := w.m.DeclareChapter(w.ctx, "link-a", w.respond.ID, w.respond.PublicKey, "", "Mill Street")
	if err != nil {
		t.Fatalf("declare on link-a: %v", err)
	}
	b, err := w.m.DeclareChapter(w.ctx, "link-b", w.respond.ID, w.respond.PublicKey, "", "Orchard Lane")
	if err != nil {
		t.Fatalf("declare on link-b after registering: %v", err)
	}
	if a.FormID == b.FormID || a.ID == b.ID {
		t.Fatalf("the two forms share a declaration: %+v %+v", a, b)
	}
}
