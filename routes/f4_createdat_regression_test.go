package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestCreateFormMintsItsOwnCreatedAt(t *testing.T) {
	fm := newTestManager(t)
	mux := mount(t, fm)

	before := time.Now().UTC().Add(-time.Minute)
	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	const forged = "1999-01-01T00:00:00Z"
	body := `{"created_at":"` + forged + `","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`
	rec := do(t, mux, http.MethodPost, "/forms", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create form: got %d, body %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaforms.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.CreatedAt == forged {
		t.Fatalf("created_at = %q, the value the request body claimed; it should be minted by the store", out.CreatedAt)
	}
	minted, err := time.Parse(mwanachamaforms.TimeLayout, out.CreatedAt)
	if err != nil {
		t.Fatalf("created_at = %q, which is not a stamp this package writes: %v", out.CreatedAt, err)
	}
	if minted.Before(before) {
		t.Fatalf("created_at = %q, which predates the request", out.CreatedAt)
	}
}
