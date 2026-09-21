package routes_test

// Board row F2: DeclareChapter rejects a respondent_id that names no Respondent
// row minted through POST {key}/respondents.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func f2Mux(fm mwanachamaforms.FormManager) *http.ServeMux {
	m := http.NewServeMux()
	for _, rt := range routes.Routes(fm, routes.ResourceNames{}) {
		m.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return m
}

// TestDeclareChapter_RejectsPhantomRespondentID: a declaration for a
// respondent_id that was never created is refused, and a real respondent's
// declaration still goes through.
func TestDeclareChapter_RejectsPhantomRespondentID(t *testing.T) {
	fm := newTestManager(t)
	m := f2Mux(fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	f, err := fm.Create(t.Context(), mwanachamaforms.Form{
		Title: "Ward Census", OriginatorChapterID: "chapter-1",
		ClosesAt: closesAt, Audience: mwanachamaforms.AudiencePublic,
		CreatedBy: "author-1",
	})
	if err != nil {
		t.Fatalf("Create form: %v", err)
	}
	_, link, err := fm.Publish(t.Context(), f.ID, "probe-link-key", "author-1")
	if err != nil {
		t.Fatalf("Publish form: %v", err)
	}

	declare := func(respondentID string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"respondent_id": respondentID, "declared_chapter_id": "chapter-99"})
		req := httptest.NewRequest(http.MethodPost, "/public-links/"+link.Key+"/declarations", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, req)
		return rec
	}

	if rec := declare("phantom-respondent-id"); rec.Code != http.StatusBadRequest {
		t.Fatalf("phantom respondent: got %d (body %s), want 400", rec.Code, rec.Body.String())
	}

	resp, err := fm.UpsertRespondent(t.Context(), mwanachamaforms.Respondent{PublicKey: "anon-key"})
	if err != nil {
		t.Fatalf("UpsertRespondent: %v", err)
	}
	if rec := declare(resp.ID); rec.Code != http.StatusCreated {
		t.Fatalf("real respondent: got %d (body %s), want 201", rec.Code, rec.Body.String())
	}
}
