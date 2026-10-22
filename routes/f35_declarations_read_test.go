package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func TestDeclarationsAreListedAndCountedPerGroupThroughTheirOwnAddresses(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)

	f, err := fm.Create(t.Context(), mwanachamaforms.Form{
		Title: "Ward Census", OriginatorChapterID: "chapter-1",
		ClosesAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), Audience: mwanachamaforms.AudiencePublic,
		CreatedBy: "author-1",
	})
	if err != nil {
		t.Fatalf("Create form: %v", err)
	}
	_, link, err := fm.Publish(t.Context(), f.ID, "census-key", "author-1", "")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	for i, ward := range []string{"ward-7", "ward-3", "ward-7"} {
		r, err := fm.UpsertRespondent(t.Context(), mwanachamaforms.Respondent{PublicKey: "anon-" + string(rune('a'+i))})
		if err != nil {
			t.Fatalf("UpsertRespondent: %v", err)
		}
		if rec := doAs(t, mux, "", http.MethodPost, "/public-links/"+link.Key+"/declarations",
			`{"respondent_id":"`+r.ID+`","declared_chapter_id":"`+ward+`"}`); rec.Code != http.StatusCreated {
			t.Fatalf("declare = %d %s", rec.Code, rec.Body.String())
		}
	}

	rec := doAs(t, mux, "operator-1", http.MethodGet, "/forms/"+f.ID+"/declarations", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list declarations = %d %s", rec.Code, rec.Body.String())
	}
	var listed []mwanachamaforms.Declaration
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || len(listed) != 3 {
		t.Fatalf("listed %s, want three declarations (%v)", rec.Body.String(), err)
	}

	rec = doAs(t, mux, "operator-1", http.MethodGet, "/forms/"+f.ID+"/declarations/count", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("count declarations = %d %s", rec.Code, rec.Body.String())
	}
	var counted []mwanachamaforms.DeclarationCount
	if err := json.Unmarshal(rec.Body.Bytes(), &counted); err != nil {
		t.Fatalf("count body %s: %v", rec.Body.String(), err)
	}
	want := []mwanachamaforms.DeclarationCount{{ChapterID: "ward-3", Count: 1}, {ChapterID: "ward-7", Count: 2}}
	if len(counted) != len(want) || counted[0] != want[0] || counted[1] != want[1] {
		t.Fatalf("counted %+v, want %+v", counted, want)
	}
}

func TestDeclarationReadsAreGatedApartAndNeverAnonymous(t *testing.T) {
	actions := map[string]string{}
	for _, rt := range routes.Routes(newTestManager(t)) {
		actions[rt.Method+" "+rt.Path] = rt.Action
	}
	if got := actions["GET /forms/{form_id}/declarations"]; got != "forms.declaration.list" {
		t.Errorf("listing declarations is gated by %q, want forms.declaration.list", got)
	}
	if got := actions["GET /forms/{form_id}/declarations/count"]; got != "forms.declaration.count" {
		t.Errorf("counting declarations is gated by %q, want forms.declaration.count", got)
	}
	for _, rt := range routes.PublicRoutes(newTestManager(t)) {
		if rt.Action == "forms.declaration.list" || rt.Action == "forms.declaration.count" {
			t.Errorf("%s is anonymous", rt.Action)
		}
	}
}
