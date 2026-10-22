package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func readLinkQuestions(t *testing.T, mux *http.ServeMux, path string) []mwanachamaforms.LinkedQuestion {
	t.Helper()
	rec := doAs(t, mux, "", http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: got %d, body %s, want 200", path, rec.Code, rec.Body.String())
	}
	var got []mwanachamaforms.LinkedQuestion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
	}
	if len(got) != 1 {
		t.Fatalf("%s: questions = %+v, want one", path, got)
	}
	return got
}

func TestF41_ASignedOutCallerReadsALinksQuestionsInAReviewedLanguage(t *testing.T) {
	fx := newLinkFixture(t)
	ctx := context.Background()
	original := readLinkQuestions(t, fx.mux, "/public-links/"+fx.linked.key+"/questions")
	put := func(role models.TranslationSubject, subject, field, lang, text string, reviewed bool) {
		t.Helper()
		if _, err := fx.fm.PutTranslation(ctx, models.Translation{
			FormID: fx.linked.form.ID, SubjectRole: role, SubjectID: subject,
			Field: field, Language: lang, Text: text, Reviewed: reviewed,
		}); err != nil {
			t.Fatalf("PutTranslation: %v", err)
		}
	}
	put(models.TranslationOfQuestion, fx.linked.question.ID, "prompt", "sw", "Barabara gani?", true)
	put(models.TranslationOfOption, original[0].Options[0].ID, "label", "sw", "Kuu", true)
	put(models.TranslationOfQuestion, fx.linked.question.ID, "prompt", "hu", "Melyik utca?", false)

	sw := readLinkQuestions(t, fx.mux, "/public-links/"+fx.linked.key+"/questions?language=sw")
	if sw[0].Prompt != "Barabara gani?" {
		t.Fatalf("sw prompt = %q, want the reviewed translation", sw[0].Prompt)
	}
	if sw[0].Options[0].Label != "Kuu" || sw[0].Options[1].Label != original[0].Options[1].Label {
		t.Fatalf("sw options = %+v, want the reviewed label and the untranslated original", sw[0].Options)
	}

	for _, lang := range []string{"hu", "fr"} {
		got := readLinkQuestions(t, fx.mux, "/public-links/"+fx.linked.key+"/questions?language="+lang)
		if got[0].Prompt != original[0].Prompt || got[0].Options[0].Label != original[0].Options[0].Label {
			t.Fatalf("%s: got %+v, want the original texts", lang, got[0])
		}
	}
}

func TestF41_ALanguageDoesNotChangeALinkQuestionsRefusal(t *testing.T) {
	fx := newLinkFixture(t)

	plain := doAs(t, fx.mux, "", http.MethodGet, "/public-links/no-such-link/questions", "")
	inSw := doAs(t, fx.mux, "", http.MethodGet, "/public-links/no-such-link/questions?language=sw", "")
	if plain.Code != http.StatusNotFound || inSw.Code != http.StatusNotFound {
		t.Fatalf("plain %d, in sw %d, want both 404", plain.Code, inSw.Code)
	}
	if plain.Body.String() != inSw.Body.String() {
		t.Fatalf("the refusals differ: plain %q, in sw %q", plain.Body.String(), inSw.Body.String())
	}
}
