package routes_test

// Pins a board row (see documentation/3. implementation/todo.md): Publish
// writes the Form's status/published_at columns and creates its PublicLink
// row as two separate, non-transactional statements. When the second write
// fails — here, because candidate_link_key collides with another form's
// already-active PublicLink.Key, which is uniquely indexed
// (gormstore/public.go's PublicLinkRow.Key) — the Form is left permanently
// stuck "open" with no PublicLink ever created for it, since Publish's own
// first branch (`case models.StatusOpen: return f, models.PublicLink{}, nil`)
// makes every future retry a silent no-op that never attempts the
// PublicLink insert again.

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

func f3Mux(fm mwanachamaforms.FormManager) *http.ServeMux {
	m := http.NewServeMux()
	for _, rt := range routes.Routes(fm, routes.ResourceNames{}) {
		m.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return m
}

// TestPublishForm_PinsColldingLinkKeyStrandsFormOpenWithNoLink pins the
// current (broken) behaviour end to end: a colliding public-link key 500s
// the HTTP request, yet the Form is already committed "open"; a subsequent
// retry (even with a fresh, non-colliding key) returns 200 success with an
// empty PublicLink and never creates one — the form is now unreachable by
// any public key, permanently. Once fixed (wrap the Form-status update and
// PublicLink insert in one transaction so a failed link insert rolls the
// status change back too, and/or map a PublicLink.Key collision to a clean
// 409 with a real sentinel instead of an opaque 500), every assertion below
// must change: the first publish attempt should refuse cleanly without
// mutating the Form, and a retry with a fresh key should succeed in
// minting a real link.
func TestPublishForm_PinsColldingLinkKeyStrandsFormOpenWithNoLink(t *testing.T) {
	fm := newTestManager(t)
	m := f3Mux(fm)
	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)

	fA, err := fm.Create(t.Context(), mwanachamaforms.Form{
		Title: "Form A", OriginatorChapterID: "chapter-1",
		ClosesAt: closesAt, Audience: mwanachamaforms.AudiencePublic, CreatedBy: "author-1",
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	if _, _, err := fm.Publish(t.Context(), fA.ID, "shared-key", "author-1"); err != nil {
		t.Fatalf("publish A: %v", err)
	}

	fB, err := fm.Create(t.Context(), mwanachamaforms.Form{
		Title: "Form B", OriginatorChapterID: "chapter-1",
		ClosesAt: closesAt, Audience: mwanachamaforms.AudiencePublic, CreatedBy: "author-2",
	})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"candidate_link_key": "shared-key", "published_by": "author-2"})
	req := httptest.NewRequest(http.MethodPost, "/forms/"+fB.ID+"/publish", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("appears fixed: publishing with a colliding link key now returns %d (body %s), not the opaque 500 this pin expects — update this test", rec.Code, rec.Body.String())
	}

	// The bug: despite the HTTP call failing, Form B's status was already
	// committed "open" by the time the PublicLink insert failed.
	fB2, err := fm.Get(t.Context(), fB.ID)
	if err != nil {
		t.Fatalf("get B: %v", err)
	}
	if fB2.Status != mwanachamaforms.StatusOpen {
		t.Fatalf("appears fixed: Form B's status is %q after a failed publish, not %q — update this test", fB2.Status, mwanachamaforms.StatusOpen)
	}

	// Retrying — even with a fresh, non-colliding key — silently no-ops via
	// Publish's own "already open" early return, and never creates a
	// PublicLink at all: Form B is now permanently public-audience, open,
	// and unreachable by any key.
	_, retryLink, err := fm.Publish(t.Context(), fB.ID, "a-perfectly-fine-fresh-key", "author-2")
	if err != nil {
		t.Fatalf("appears fixed: retrying Publish on the stranded form now errors (%v) instead of silently no-opping — update this test", err)
	}
	if retryLink.Key != "" {
		t.Fatalf("appears fixed: retry now mints a real PublicLink (key %q) — update this test to assert recovery instead of the stranded state", retryLink.Key)
	}
}
