package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestF41_TranslationsArePutAndReadThroughTheirOwnAddresses(t *testing.T) {
	fm := newTestManager(t)
	mux := mount(t, fm)
	f := seedForm(t, fm)

	put := func(lang, text string, reviewed bool) {
		t.Helper()
		body := `{"form_id":"another-form","subject_role":"form","subject_id":"` + f.ID +
			`","field":"title","language":"` + lang + `","text":"` + text + `","reviewed":` +
			map[bool]string{true: "true", false: "false"}[reviewed] + `}`
		rec := do(t, mux, http.MethodPut, "/forms/"+f.ID+"/translations", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("put %s = %d %s", lang, rec.Code, rec.Body.String())
		}
		var got models.Translation
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.FormID != f.ID {
			t.Fatalf("put %s answered %s, want it filed under the addressed form (%v)", lang, rec.Body.String(), err)
		}
	}
	put("hu", "Kérdőív", true)
	put("sw", "Dodoso", false)

	rec := do(t, mux, http.MethodGet, "/forms/"+f.ID+"/translations", "")
	var all []models.Translation
	if err := json.Unmarshal(rec.Body.Bytes(), &all); rec.Code != http.StatusOK || err != nil || len(all) != 2 {
		t.Fatalf("list = %d %s, want both texts", rec.Code, rec.Body.String())
	}

	rec = do(t, mux, http.MethodGet, "/forms/"+f.ID+"/translations/reviewed?language=sw", "")
	var reviewed []models.Translation
	if err := json.Unmarshal(rec.Body.Bytes(), &reviewed); rec.Code != http.StatusOK || err != nil || len(reviewed) != 0 {
		t.Fatalf("reviewed sw = %d %s, want none served", rec.Code, rec.Body.String())
	}
	rec = do(t, mux, http.MethodGet, "/forms/"+f.ID+"/translations/reviewed?language=hu", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &reviewed); rec.Code != http.StatusOK || err != nil || len(reviewed) != 1 || reviewed[0].Text != "Kérdőív" {
		t.Fatalf("reviewed hu = %d %s, want the one reviewed text", rec.Code, rec.Body.String())
	}

	rec = do(t, mux, http.MethodGet, "/forms/"+f.ID+"/languages", "")
	var langs []string
	if err := json.Unmarshal(rec.Body.Bytes(), &langs); rec.Code != http.StatusOK || err != nil || len(langs) != 1 || langs[0] != "hu" {
		t.Fatalf("languages = %d %s, want [hu]", rec.Code, rec.Body.String())
	}

	if rec := do(t, mux, http.MethodPut, "/forms/missing/translations",
		`{"subject_role":"form","subject_id":"missing","field":"title","language":"hu","text":"x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("put on a missing form = %d %s, want 404", rec.Code, rec.Body.String())
	}
}
