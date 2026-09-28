package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestCreateForm_IgnoresCallerSuppliedID(t *testing.T) {
	fm := newTestManager(t)
	mux := mount(t, fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	const wanted = "attacker-chosen-form-id"
	body := `{"id":"` + wanted + `","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`
	rec := do(t, mux, http.MethodPost, "/forms", body)

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
	mux := mount(t, fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	body := `{"id":"attacker-chosen-form-id-2","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`

	firstRec := do(t, mux, http.MethodPost, "/forms", body)
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("first create: got %d, body %s", firstRec.Code, firstRec.Body.String())
	}

	secondRec := do(t, mux, http.MethodPost, "/forms", body)
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
