package routes_test

// Board row F1: CreateForm always mints its own id, so a caller-supplied "id"
// is ignored and re-posting the same body cannot collide on the primary key.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func TestCreateForm_IgnoresCallerSuppliedID(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.CreateForm(fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	const wanted = "attacker-chosen-form-id"
	body := `{"id":"` + wanted + `","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`
	req := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create form: got %d, body %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaforms.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ID == wanted {
		t.Fatalf("a caller-supplied id was honoured (got %q) — the server must mint its own", out.ID)
	}
}

func TestCreateForm_DuplicateCallerSuppliedIDMintsDistinctIDs(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.CreateForm(fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	body := `{"id":"attacker-chosen-form-id-2","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`

	first := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(body))
	firstRec := httptest.NewRecorder()
	handler(firstRec, first)
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("first create: got %d, body %s", firstRec.Code, firstRec.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(body))
	secondRec := httptest.NewRecorder()
	handler(secondRec, second)
	if secondRec.Code != http.StatusCreated {
		t.Fatalf("second create: got %d, body %s", secondRec.Code, secondRec.Body.String())
	}
	var a, b mwanachamaforms.Form
	if err := json.Unmarshal(firstRec.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &b); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if a.ID == b.ID || a.ID == "attacker-chosen-form-id-2" || b.ID == "attacker-chosen-form-id-2" {
		t.Fatalf("expected two distinct server-minted ids, got %q and %q", a.ID, b.ID)
	}
}
