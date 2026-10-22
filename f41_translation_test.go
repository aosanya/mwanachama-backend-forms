package mwanachamaforms_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func translator(t *testing.T) (mwanachamaforms.FormManager, mwanachamaforms.Translator) {
	t.Helper()
	mgr := newTestManager(t)
	tr, ok := mgr.(mwanachamaforms.Translator)
	if !ok {
		t.Fatal("the form manager does not translate")
	}
	return mgr, tr
}

func TestF41_OnlyReviewedTranslationsAreServed(t *testing.T) {
	ctx := context.Background()
	mgr, tr := translator(t)
	f := newDraftForm(t, mgr)
	put := func(lang, text string, reviewed bool) {
		t.Helper()
		_, err := tr.PutTranslation(ctx, models.Translation{
			FormID: f.ID, SubjectRole: models.TranslationOfForm, SubjectID: f.ID,
			Field: "title", Language: lang, Text: text, Reviewed: reviewed,
		})
		if err != nil {
			t.Fatalf("PutTranslation %s: %v", lang, err)
		}
	}
	put("sw", "Ukaguzi wa Afya", true)
	put("hu", "Egészségügyi felmérés", false)

	served, err := tr.ReviewedTranslations(ctx, f.ID, "hu")
	if err != nil || len(served) != 0 {
		t.Fatalf("an unreviewed translation was served: %v %v", served, err)
	}
	langs, err := tr.TranslationLanguages(ctx, f.ID)
	if err != nil || !reflect.DeepEqual(langs, []string{"sw"}) {
		t.Fatalf("languages = %v %v, want [sw]", langs, err)
	}

	put("hu", "Egészségügyi felmérés", true)
	served, _ = tr.ReviewedTranslations(ctx, f.ID, "hu")
	if len(served) != 1 || served[0].Text != "Egészségügyi felmérés" {
		t.Fatalf("reviewed translation not served: %v", served)
	}
	langs, _ = tr.TranslationLanguages(ctx, f.ID)
	if !reflect.DeepEqual(langs, []string{"hu", "sw"}) {
		t.Fatalf("languages = %v, want [hu sw]", langs)
	}
}

func TestF41_ACorrectionReplacesAndNeedsReviewAgain(t *testing.T) {
	ctx := context.Background()
	mgr, tr := translator(t)
	f := newDraftForm(t, mgr)
	tl := models.Translation{
		FormID: f.ID, SubjectRole: models.TranslationOfQuestion, SubjectID: "q-1",
		Field: "prompt", Language: "sw", Text: "Swali", Reviewed: true,
	}
	if _, err := tr.PutTranslation(ctx, tl); err != nil {
		t.Fatal(err)
	}
	tl.Text, tl.Reviewed = "Swali jipya", false
	if _, err := tr.PutTranslation(ctx, tl); err != nil {
		t.Fatal(err)
	}
	all, _ := tr.ListTranslations(ctx, f.ID)
	if len(all) != 1 || all[0].Text != "Swali jipya" || all[0].Reviewed {
		t.Fatalf("correction did not replace: %v", all)
	}
	if served, _ := tr.ReviewedTranslations(ctx, f.ID, "sw"); len(served) != 0 {
		t.Fatalf("an unreviewed correction was served: %v", served)
	}
}

func TestF41_ATranslationNeedsARealFormAndSubject(t *testing.T) {
	ctx := context.Background()
	mgr, tr := translator(t)
	f := newDraftForm(t, mgr)
	_, err := tr.PutTranslation(ctx, models.Translation{
		FormID: "missing", SubjectRole: models.TranslationOfForm, SubjectID: "missing", Field: "title", Language: "sw",
	})
	if !errors.Is(err, mwanachamaforms.ErrFormNotFound) {
		t.Fatalf("missing form: %v", err)
	}
	_, err = tr.PutTranslation(ctx, models.Translation{
		FormID: f.ID, SubjectRole: "answer", SubjectID: f.ID, Field: "title", Language: "sw",
	})
	if !errors.Is(err, mwanachamaforms.ErrInvalidReference) {
		t.Fatalf("bad subject role: %v", err)
	}
}
