package routes_test

// Pins a board row (see documentation/3. implementation/todo.md): CreateForm
// honors a caller-supplied "id" instead of always minting its own, and a
// duplicate id 500s instead of a clean conflict. Same bug class found this
// same fleet sweep in mwanachama-backend-taskmanager, -assetmanager,
// -accounting and -custody — this repo has neither the ID-clearing fix nor
// a duplicate-id sentinel (unlike mwanachama-backend-actor/-comm, which
// deliberately allow a caller-chosen id but classify a collision into a
// clean ErrDuplicateID/ErrConflict).

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

// Once fixed (form_impl.go's Create clearing f.ID before use, the same fix
// applied elsewhere in this fleet's sweep), out.ID must NOT equal the
// caller-supplied value.
func TestCreateForm_PinsCallerSuppliedIDIsHonored(t *testing.T) {
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
	if out.ID != wanted {
		t.Fatalf("appears fixed: CreateForm no longer honors a caller-supplied id (got %q) — update this pin to assert the new server-minted-id behavior instead", out.ID)
	}
}

// Once fixed (either the id-spoofing gap closes, or a genuine collision maps
// to a 409 the way mwanachama-backend-actor's ErrDuplicateID/
// mwanachama-backend-comm's ErrConflict already do), the second POST below
// must NOT be 500.
func TestCreateForm_PinsDuplicateCallerSuppliedIDReturns500NotConflict(t *testing.T) {
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
	if secondRec.Code != http.StatusInternalServerError {
		t.Fatalf("appears fixed: duplicate id now returns %d (body %s), not the unmapped 500 this pin expects", secondRec.Code, secondRec.Body.String())
	}
}
