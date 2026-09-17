package routes_test

// Pins a board row (see documentation/3. implementation/todo.md): the
// public POST {key}/declarations route (DeclareChapter, backed by
// FormManager.Declare) never checks that the caller-supplied
// "respondent_id" names a real Respondent row that was ever actually
// minted through POST {key}/respondents — Declare's own upsert query only
// matches on (RespondentID, FormID), so any string at all is accepted and
// written as a live Declaration row.

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

// TestDeclareChapter_PinsPhantomRespondentIDIsAccepted pins the current
// (broken) behaviour: DeclareChapter accepts and stores a declaration for
// a respondent_id that was never created by POST {key}/respondents — no
// real respondent exists behind it at all. Once fixed (Declare, or the
// DeclareChapter handler, validating the respondent exists before writing),
// the assertion below expecting 201 must change to expect a rejection.
func TestDeclareChapter_PinsPhantomRespondentIDIsAccepted(t *testing.T) {
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
	if link.Key == "" {
		t.Fatalf("Publish did not mint a public link for a public-audience form")
	}

	// No POST {key}/respondents call ever happens — "phantom-respondent-id"
	// names no Respondent row created through this form's own public flow.
	body, _ := json.Marshal(map[string]string{
		"respondent_id":       "phantom-respondent-id",
		"declared_chapter_id": "chapter-99",
	})
	req := httptest.NewRequest(http.MethodPost, "/public-links/"+link.Key+"/declarations", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("appears fixed: declaring a phantom respondent_id now returns %d (body %s), not the 201 this pin expects — update this test to assert the rejection instead", rec.Code, rec.Body.String())
	}
	var out mwanachamaforms.Declaration
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.RespondentID != "phantom-respondent-id" {
		t.Fatalf("got RespondentID %q, want the phantom id echoed back", out.RespondentID)
	}

	// Confirm the phantom declaration is now a live, readable row: a second
	// identical call overwrites in place rather than erroring on "unknown
	// respondent" — reinforcing that nothing downstream of Declare treats
	// RespondentID as a real foreign key either.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/public-links/"+link.Key+"/declarations", bytes.NewReader(body))
	m.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("second phantom declaration: got %d, body %s", rec2.Code, rec2.Body.String())
	}
}
