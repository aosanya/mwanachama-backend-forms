package routes_test

// Board row F3: Publish writes the Form's status and its PublicLink in one
// transaction, so a colliding link key fails cleanly (409) and leaves the Form
// unpublished and retryable.

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

func TestPublishForm_CollidingLinkKeyLeavesFormRetryable(t *testing.T) {
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
	if rec.Code != http.StatusConflict {
		t.Fatalf("colliding link key: got %d (body %s), want 409", rec.Code, rec.Body.String())
	}

	fB2, err := fm.Get(t.Context(), fB.ID)
	if err != nil {
		t.Fatalf("get B: %v", err)
	}
	if fB2.Status == mwanachamaforms.StatusOpen {
		t.Fatalf("Form B was left %q after a failed publish", fB2.Status)
	}

	_, retryLink, err := fm.Publish(t.Context(), fB.ID, "a-perfectly-fine-fresh-key", "author-2")
	if err != nil {
		t.Fatalf("retry with a fresh key: %v", err)
	}
	if retryLink.Key != "a-perfectly-fine-fresh-key" {
		t.Fatalf("retry minted link key %q, want the fresh key", retryLink.Key)
	}
}
