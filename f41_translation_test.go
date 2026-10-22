package mwanachamaforms_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

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
	q := addTranslatableQuestion(t, mgr, f.ID)
	tl := models.Translation{
		FormID: f.ID, SubjectRole: models.TranslationOfQuestion, SubjectID: q.ID,
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

func TestF41_AFormKeepsTheLanguageItWasWrittenIn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Kérdőív", ClosesAt: futureRFC3339(time.Hour), Language: "hu"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := mgr.Get(ctx, f.ID)
	if err != nil || got.Language != "hu" {
		t.Fatalf("Get = %+v %v, want the language it was written in", got, err)
	}
}

func addTranslatableQuestion(t *testing.T, mgr mwanachamaforms.FormManager, formID string) mwanachamaforms.Question {
	t.Helper()
	q, _, err := mgr.AddQuestion(context.Background(), mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: formID, AnswerType: mwanachamaforms.AnswerSingleChoice, Prompt: "Which ward?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	return q
}

func TestF41_ATranslationSubjectMustBelongToTheNamedForm(t *testing.T) {
	ctx := context.Background()
	mgr, tr := translator(t)
	mine := newDraftForm(t, mgr)
	other := newDraftForm(t, mgr)
	myQuestion := addTranslatableQuestion(t, mgr, mine.ID)
	otherQuestion := addTranslatableQuestion(t, mgr, other.ID)
	myOption, err := mgr.AddOption(ctx, myQuestion.ID, "North", "")
	if err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	otherOption, err := mgr.AddOption(ctx, otherQuestion.ID, "South", "")
	if err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	put := func(role models.TranslationSubject, subject string) error {
		_, err := tr.PutTranslation(ctx, models.Translation{
			FormID: mine.ID, SubjectRole: role, SubjectID: subject,
			Field: "text", Language: "sw", Text: "x",
		})
		return err
	}
	for name, err := range map[string]error{
		"own question": put(models.TranslationOfQuestion, myQuestion.ID),
		"own option":   put(models.TranslationOfOption, myOption.Option.ID),
		"own form":     put(models.TranslationOfForm, mine.ID),
	} {
		if err != nil {
			t.Errorf("%s refused: %v", name, err)
		}
	}
	for name, err := range map[string]error{
		"another form's question":     put(models.TranslationOfQuestion, otherQuestion.ID),
		"another form's option":       put(models.TranslationOfOption, otherOption.Option.ID),
		"another form as the subject": put(models.TranslationOfForm, other.ID),
		"an option named as question": put(models.TranslationOfQuestion, myOption.Option.ID),
		"a question named as option":  put(models.TranslationOfOption, myQuestion.ID),
		"an invented question":        put(models.TranslationOfQuestion, "q-invented"),
	} {
		if !errors.Is(err, mwanachamaforms.ErrInvalidReference) {
			t.Errorf("%s: err = %v, want ErrInvalidReference", name, err)
		}
	}
	all, _ := tr.ListTranslations(ctx, mine.ID)
	if len(all) != 3 {
		t.Fatalf("stored %d translations, want the 3 accepted", len(all))
	}
}
