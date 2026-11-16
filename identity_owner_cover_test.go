package mwanachamaforms_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/identity"
)

type identityOwner interface {
	StableIdentity(ctx context.Context, object, id string) (string, error)
	Resolve(ctx context.Context, raw string) (string, string, error)
}

func ownerOf(t *testing.T, m forms.FormManager) identityOwner {
	t.Helper()
	o, ok := m.(identityOwner)
	if !ok {
		t.Fatal("the form manager does not own identities")
	}
	return o
}

func TestAFormHasAStableIdentityThatResolvesBackToIt(t *testing.T) {
	m := gatedManager(t, gateFor{})
	r := openableRound(t, m, personLandsIn())
	owner := ownerOf(t, m)
	ctx := context.Background()

	raw, err := owner.StableIdentity(ctx, "form", r.form.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	if !strings.HasPrefix(raw, "forms/test/") {
		t.Fatalf("the identity does not name its module and instance: %q", raw)
	}
	object, id, err := owner.Resolve(ctx, raw)
	if err != nil || object != "form" || id != r.form.ID {
		t.Fatalf("Resolve = %q %q %v", object, id, err)
	}
}

func TestAnObjectWithoutAStableIdentityIsRefused(t *testing.T) {
	m := gatedManager(t, gateFor{})
	if _, err := ownerOf(t, m).StableIdentity(context.Background(), "answer", "a-1"); !errors.Is(err, forms.ErrNoStableIdentity) {
		t.Fatalf("StableIdentity of an answer = %v, want ErrNoStableIdentity", err)
	}
}

func TestAnUnknownFormHasNoIdentity(t *testing.T) {
	m := gatedManager(t, gateFor{})
	if _, err := ownerOf(t, m).StableIdentity(context.Background(), "form", "missing"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("StableIdentity of a missing form = %v, want ErrNotFound", err)
	}
}

func TestAQuestionKeepsItsIdentityAcrossVersionsAndResolvesToTheCurrentOne(t *testing.T) {
	m := gatedManager(t, gateFor{})
	m.UseRecordWriters(writersFor("catalog", &collectingWriter{}))
	r := openableRound(t, m, personLandsIn())
	owner := ownerOf(t, m)
	ctx := context.Background()
	if _, _, err := m.Publish(ctx, r.form.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	first, err := owner.StableIdentity(ctx, "question", r.name.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	object, id, err := owner.Resolve(ctx, first)
	if err != nil || object != "question" || id != r.name.ID {
		t.Fatalf("Resolve before a new version = %q %q %v", object, id, err)
	}

	next, err := m.VersionQuestion(ctx, r.name.ID, "author-1", models.QuestionDraft{Question: models.Question{Prompt: "Full legal name", AnswerType: models.AnswerFreeText}})
	if err != nil {
		t.Fatalf("VersionQuestion: %v", err)
	}

	second, err := owner.StableIdentity(ctx, "question", next.Question.ID)
	if err != nil {
		t.Fatalf("StableIdentity of the new version: %v", err)
	}
	if second != first {
		t.Fatalf("a new version changed the identity: %q then %q", first, second)
	}
	object, id, err = owner.Resolve(ctx, first)
	if err != nil || object != "question" || id != next.Question.ID {
		t.Fatalf("Resolve after a new version = %q %q %v, want the new version %s", object, id, err, next.Question.ID)
	}
}

func TestResolveRefusesWhatThisModuleDoesNotOwn(t *testing.T) {
	m := gatedManager(t, gateFor{})
	owner := ownerOf(t, m)
	ctx := context.Background()

	if _, _, err := owner.Resolve(ctx, "not-an-identity"); !errors.Is(err, identity.ErrInvalidRaw) {
		t.Fatalf("Resolve of garbage = %v, want ErrInvalidRaw", err)
	}
	if _, _, err := owner.Resolve(ctx, "people/test/p1"); !errors.Is(err, identity.ErrNotOwned) {
		t.Fatalf("Resolve of another module's identity = %v, want ErrNotOwned", err)
	}
	if _, _, err := owner.Resolve(ctx, "forms/other/f1"); !errors.Is(err, identity.ErrNotOwned) {
		t.Fatalf("Resolve of another instance's identity = %v, want ErrNotOwned", err)
	}
	if _, _, err := owner.Resolve(ctx, "forms/test/nothing"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("Resolve of an unknown identity = %v, want ErrNotFound", err)
	}
}
