package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/identity"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

var _ identity.Owner = mwanachamaforms.FormManager(nil)

func publishedFormWithAQuestion(t *testing.T, mgr mwanachamaforms.FormManager) (mwanachamaforms.Form, mwanachamaforms.Question) {
	t.Helper()
	ctx := context.Background()
	f := newDraftForm(t, mgr)
	q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How was the month?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := mgr.Publish(ctx, f.ID, "", "admin", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f, q
}

func versionOf(t *testing.T, mgr mwanachamaforms.FormManager, questionID, prompt string) mwanachamaforms.Question {
	t.Helper()
	result, err := mgr.VersionQuestion(context.Background(), questionID, "editor", mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{Prompt: prompt}})
	if err != nil {
		t.Fatalf("VersionQuestion: %v", err)
	}
	return result.Question
}

func TestStableIdentityIsTheRawFormsInstanceUuidOfAFormAndOfAQuestion(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := publishedFormWithAQuestion(t, mgr)

	for object, id := range map[string]string{"form": f.ID, "question": q.ID} {
		got, err := mgr.StableIdentity(ctx, object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		if want := "forms/test/" + id; got != want {
			t.Errorf("StableIdentity(%s) = %q, want %q", object, got, want)
		}
	}
}

func TestResolveGivesBackTheObjectAndIdStableIdentityWasAskedFor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := publishedFormWithAQuestion(t, mgr)

	for object, id := range map[string]string{"form": f.ID, "question": q.ID} {
		raw, err := mgr.StableIdentity(ctx, object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		gotObject, gotID, err := mgr.Resolve(ctx, raw)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", raw, err)
		}
		if gotObject != object || gotID != id {
			t.Errorf("Resolve(%q) = (%q, %q), want (%q, %q)", raw, gotObject, gotID, object, id)
		}
	}
}

func TestAQuestionKeepsItsIdentityThroughEveryVersionWhileTheRowIdChanges(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, first := publishedFormWithAQuestion(t, mgr)
	firstRaw, err := mgr.StableIdentity(ctx, "question", first.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	second := versionOf(t, mgr, first.ID, "Clearer wording")
	third := versionOf(t, mgr, second.ID, "Clearest wording")
	if second.ID == first.ID || third.ID == second.ID {
		t.Fatalf("precondition: a version is a new row")
	}

	for _, v := range []mwanachamaforms.Question{second, third} {
		raw, err := mgr.StableIdentity(ctx, "question", v.ID)
		if err != nil {
			t.Fatalf("StableIdentity(version %d): %v", v.Version, err)
		}
		if raw != firstRaw {
			t.Errorf("version %d is %q, the first %q; a new version must keep the identity", v.Version, raw, firstRaw)
		}
	}

	object, id, err := mgr.Resolve(ctx, firstRaw)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if object != "question" || id != third.ID {
		t.Errorf("Resolve(%q) = (%q, %q), want the current version (question, %q)", firstRaw, object, id, third.ID)
	}
}

func TestAVersionsOwnRowIdIsNotAnIdentityToResolve(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, first := publishedFormWithAQuestion(t, mgr)
	second := versionOf(t, mgr, first.ID, "Clearer wording")

	if _, _, err := mgr.Resolve(ctx, "forms/test/"+second.ID); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("a later version's row id as a raw: want ErrNotFound, got %v", err)
	}
	if _, _, err := mgr.Resolve(ctx, "forms/test/"+f.ID); err != nil {
		t.Errorf("the form is unaffected by a question's versions: %v", err)
	}
}

func TestADeletedFormAndARemovedQuestionAreConfirmedMisses(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f := newDraftForm(t, mgr)
	q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "Anything?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	formRaw, err := mgr.StableIdentity(ctx, "form", f.ID)
	if err != nil {
		t.Fatalf("StableIdentity(form): %v", err)
	}
	questionRaw, err := mgr.StableIdentity(ctx, "question", q.ID)
	if err != nil {
		t.Fatalf("StableIdentity(question): %v", err)
	}
	if err := mgr.Delete(ctx, f.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	for _, raw := range []string{formRaw, questionRaw} {
		if _, _, err := mgr.Resolve(ctx, raw); !errors.Is(err, identity.ErrNotFound) {
			t.Errorf("Resolve(%q) after the form was deleted: want ErrNotFound, got %v", raw, err)
		}
	}
	if _, err := mgr.StableIdentity(ctx, "form", f.ID); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("StableIdentity of a deleted form: want ErrNotFound, got %v", err)
	}
}

func TestResolveNamesAMissAndARawItDoesNotOwnApart(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, _ := publishedFormWithAQuestion(t, mgr)
	cases := []struct {
		raw  string
		want error
	}{
		{"forms/test/00000000-0000-0000-0000-000000000000", identity.ErrNotFound},
		{"git/test/" + f.ID, identity.ErrNotOwned},
		{"forms/elsewhere/" + f.ID, identity.ErrNotOwned},
		{"forms/test", identity.ErrInvalidRaw},
		{"", identity.ErrInvalidRaw},
	}
	for _, c := range cases {
		if _, _, err := mgr.Resolve(ctx, c.raw); !errors.Is(err, c.want) {
			t.Errorf("Resolve(%q): want %v, got %v", c.raw, c.want, err)
		}
	}
}

func TestStableIdentityNamesAMissAndAnObjectWithNoIdentityApart(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, _ := publishedFormWithAQuestion(t, mgr)

	for _, object := range []string{"form", "question"} {
		if _, err := mgr.StableIdentity(ctx, object, "no-such-row"); !errors.Is(err, identity.ErrNotFound) {
			t.Errorf("a missing %s: want ErrNotFound, got %v", object, err)
		}
	}
	for _, object := range []string{"question_option", "target", "approval", "answer", "submission", "public_link", "no_such_object"} {
		if _, err := mgr.StableIdentity(ctx, object, f.ID); !errors.Is(err, mwanachamaforms.ErrNoStableIdentity) {
			t.Errorf("%s: want ErrNoStableIdentity, got %v", object, err)
		}
	}
}
