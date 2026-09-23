package routes_test

// Board row F4: unlike CreateForm's id (see w_caller_supplied_id_test.go's
// F1 pin — already server-minted correctly), CreateForm's created_at is
// only defaulted "if f.CreatedAt == \"\"" (form_impl.go), so a caller can
// forge the audit timestamp on an otherwise ordinary POST /forms.

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

// TestPinsF4_CreateFormHonoursCallerSuppliedCreatedAt pins a known defect
// (board row F4): CreateForm clears a caller-supplied id (F1 already
// covers that) but never clears created_at, so a caller can backdate a
// Form's own audit timestamp. Must go RED once F4's fix lands (created_at
// should be minted unconditionally, dropping the "if == \"\"" guard, the
// same fix ACT4/AG21 apply for the identical shape elsewhere in the
// fleet) — flip the assertion below to check the response no longer
// echoes the caller's forged timestamp.
func TestPinsF4_CreateFormHonoursCallerSuppliedCreatedAt(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.CreateForm(fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	const forgedCreatedAt = "1999-01-01T00:00:00Z"
	body := `{"created_at":"` + forgedCreatedAt + `","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`
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
	// KNOWN DEFECT (F4): the caller's forged created_at is echoed back
	// verbatim instead of being server-minted.
	if out.CreatedAt != forgedCreatedAt {
		t.Fatalf("pin: current (buggy) behaviour honours the caller's created_at verbatim; got %q, want %q — F4 may already be fixed", out.CreatedAt, forgedCreatedAt)
	}
}
