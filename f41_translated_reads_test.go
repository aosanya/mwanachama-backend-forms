package mwanachamaforms_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func translateLinkedSurvey(t *testing.T, mgr mwanachamaforms.FormManager, s linkedSurvey) models.QuestionOption {
	t.Helper()
	ctx := context.Background()
	options, err := mgr.ListOptions(ctx, s.choice.ID)
	if err != nil || len(options) != 2 {
		t.Fatalf("ListOptions: %v %v", options, err)
	}
	put := func(role models.TranslationSubject, subject, field, lang, text string, reviewed bool) {
		t.Helper()
		if _, err := mgr.PutTranslation(ctx, models.Translation{
			FormID: s.form.ID, SubjectRole: role, SubjectID: subject,
			Field: field, Language: lang, Text: text, Reviewed: reviewed,
		}); err != nil {
			t.Fatalf("PutTranslation: %v", err)
		}
	}
	put(models.TranslationOfForm, s.form.ID, "title", "sw", "Taa za barabarani", true)
	put(models.TranslationOfQuestion, s.choice.ID, "prompt", "sw", "Barabara gani?", true)
	put(models.TranslationOfQuestion, s.text.ID, "prompt", "sw", "Nini ni giza?", false)
	put(models.TranslationOfOption, options[0].ID, "label", "sw", "Kuu", true)
	put(models.TranslationOfQuestion, s.choice.ID, "prompt", "hu", "Melyik utca?", false)
	return options[0]
}

func TestF41_AReadInALanguageServesItsReviewedTexts(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	s := composeLinkedSurvey(t, mgr, "link-a")
	main := translateLinkedSurvey(t, mgr, s)

	f, err := mgr.GetIn(ctx, s.form.ID, "sw")
	if err != nil || f.Title != "Taa za barabarani" {
		t.Fatalf("GetIn sw = %q %v", f.Title, err)
	}
	qs, err := mgr.ListQuestionsIn(ctx, s.form.ID, "sw")
	if err != nil {
		t.Fatal(err)
	}
	prompts := map[string]string{}
	for _, q := range qs {
		prompts[q.ID] = q.Prompt
	}
	if prompts[s.choice.ID] != "Barabara gani?" {
		t.Fatalf("reviewed prompt not served: %v", prompts)
	}
	if prompts[s.text.ID] != "What is dark?" {
		t.Fatalf("an unreviewed prompt was served: %v", prompts)
	}
	opts, err := mgr.ListOptionsIn(ctx, s.choice.ID, "sw")
	if err != nil || len(opts) != 2 || opts[0].ID != main.ID || opts[0].Label != "Kuu" || opts[1].Label != "Market" {
		t.Fatalf("ListOptionsIn sw = %+v %v", opts, err)
	}
	linked, err := mgr.PublicLinkQuestionsIn(ctx, "link-a", "sw")
	if err != nil || len(linked) != 3 {
		t.Fatalf("PublicLinkQuestionsIn: %v %v", linked, err)
	}
	if linked[0].Prompt != "What is dark?" || linked[1].Prompt != "Barabara gani?" || linked[1].Options[0].Label != "Kuu" {
		t.Fatalf("link read in sw = %+v", linked)
	}
}

func TestF41_AnUnknownOrUnreviewedLanguageServesTheOriginal(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	s := composeLinkedSurvey(t, mgr, "link-a")
	translateLinkedSurvey(t, mgr, s)
	original, err := mgr.PublicLinkQuestions(ctx, "link-a")
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"", "hu", "fr"} {
		f, err := mgr.GetIn(ctx, s.form.ID, lang)
		if err != nil || f.Title != "Street lighting" {
			t.Fatalf("GetIn %q = %q %v", lang, f.Title, err)
		}
		opts, err := mgr.ListOptionsIn(ctx, s.choice.ID, lang)
		if err != nil || opts[0].Label != "Main" {
			t.Fatalf("ListOptionsIn %q = %+v %v", lang, opts, err)
		}
		linked, err := mgr.PublicLinkQuestionsIn(ctx, "link-a", lang)
		if err != nil || !reflect.DeepEqual(linked, original) {
			t.Fatalf("link read in %q = %+v, want the original %+v", lang, linked, original)
		}
	}
}

func TestF41_ALanguageDoesNotChangeARefusal(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	s := composeLinkedSurvey(t, mgr, "link-a")
	translateLinkedSurvey(t, mgr, s)
	setColumn(t, db, "form", "id", s.form.ID, "audience", "member")

	stranger := mwanachamaforms.OnlyReached(context.Background(), "stranger")
	_, plainErr := mgr.Get(stranger, s.form.ID)
	_, err := mgr.GetIn(stranger, s.form.ID, "sw")
	if !errors.Is(plainErr, mwanachamaforms.ErrFormNotFound) || err == nil || err.Error() != plainErr.Error() {
		t.Fatalf("GetIn refused as %v, Get as %v", err, plainErr)
	}
	qs, err := mgr.ListQuestionsIn(stranger, s.form.ID, "sw")
	if err != nil || len(qs) != 0 {
		t.Fatalf("an unreached form's prompts were read in sw: %v %v", qs, err)
	}
	opts, err := mgr.ListOptionsIn(stranger, s.choice.ID, "sw")
	if err != nil || len(opts) != 0 {
		t.Fatalf("an unreached form's choices were read in sw: %v %v", opts, err)
	}
	linked, err := mgr.PublicLinkQuestionsIn(context.Background(), "link-a", "sw")
	if !errors.Is(err, mwanachamaforms.ErrLinkNotFound) || err.Error() != mwanachamaforms.ErrLinkNotFound.Error() || linked != nil {
		t.Fatalf("link read in sw = %v %v, want exactly ErrLinkNotFound", linked, err)
	}
}
