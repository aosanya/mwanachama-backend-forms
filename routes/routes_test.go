package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func newTestManager(t *testing.T) mwanachamaforms.FormManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := mwanachamaforms.DefaultTableNames("routes_test")
	if err := mwanachamaforms.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	fm, err := mwanachamaforms.NewFormManager(db, tables)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	return fm
}

func withPathValue(req *http.Request, kv ...string) *http.Request {
	for i := 0; i+1 < len(kv); i += 2 {
		req.SetPathValue(kv[i], kv[i+1])
	}
	return req
}

func patterns(rts []routes.Route, prefix string) []string {
	out := make([]string, len(rts))
	for i, rt := range rts {
		out[i] = rt.Pattern(prefix)
	}
	return out
}

func assertPatterns(t *testing.T, got []routes.Route, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d routes, want %d: %v", len(got), len(want), patterns(got, ""))
	}
	for i, p := range patterns(got, "") {
		if p != want[i] {
			t.Fatalf("route %d: got %q, want %q", i, p, want[i])
		}
	}
}

func TestFormRoutes_DefaultResourceIsForms(t *testing.T) {
	fm := newTestManager(t)
	rts := routes.FormRoutes(fm, routes.ResourceNames{})
	assertPatterns(t, rts, []string{
		"POST /forms", "GET /forms", "GET /forms/{formID}", "PATCH /forms/{formID}", "DELETE /forms/{formID}",
		"POST /forms/{formID}/submit", "POST /forms/{formID}/withdraw", "POST /forms/{formID}/publish", "POST /forms/{formID}/close",
	})
}

func TestRoutes_ConcatenatesEveryBuilder(t *testing.T) {
	fm := newTestManager(t)
	all := routes.Routes(fm, routes.ResourceNames{})
	want := 9 + 4 + 6 + 3 + 3 + 2 + 3 // Form + Approval + Question + Target + Propagation + Answer + PublicLink
	if len(all) != want {
		t.Fatalf("got %d routes, want %d: %v", len(all), want, patterns(all, ""))
	}
}

func TestRoute_PatternWithPrefix(t *testing.T) {
	fm := newTestManager(t)
	rts := routes.FormRoutes(fm, routes.ResourceNames{})
	if got := rts[0].Pattern("/v1"); got != "POST /v1/forms" {
		t.Fatalf("got %q", got)
	}
}

func TestCreateForm_Handler(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.CreateForm(fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	body := `{"title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`
	req := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaforms.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Title != "Chapter Census" || out.Status != mwanachamaforms.StatusDraft {
		t.Fatalf("unexpected form: %+v", out)
	}
}

func TestCreateForm_MissingTitle(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.CreateForm(fm)

	req := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(`{"closes_at":"2099-01-01T00:00:00Z"}`))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func newFormViaHandler(t *testing.T, fm mwanachamaforms.FormManager) mwanachamaforms.Form {
	t.Helper()
	f, err := fm.Create(context.Background(), mwanachamaforms.Form{
		Title: "Seed", OriginatorChapterID: "chapter-1", ClosesAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	return f
}

func TestGetForm_NotFound(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.GetForm(fm)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/forms/nope", nil), "formID", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPublishForm_Handler(t *testing.T) {
	fm := newTestManager(t)
	f := newFormViaHandler(t, fm)
	handler := routes.PublishForm(fm)

	req := withPathValue(httptest.NewRequest(http.MethodPost, "/forms/"+f.ID+"/publish", strings.NewReader(`{"published_by":"admin"}`)), "formID", f.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestAddTarget_DuplicateConflict(t *testing.T) {
	fm := newTestManager(t)
	f := newFormViaHandler(t, fm)
	handler := routes.AddTarget(fm)

	req1 := withPathValue(httptest.NewRequest(http.MethodPost, "/forms/"+f.ID+"/targets", strings.NewReader(`{"chapter_id":"chapter-2"}`)), "formID", f.ID)
	rec1 := httptest.NewRecorder()
	handler(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec1.Code, rec1.Body.String())
	}

	req2 := withPathValue(httptest.NewRequest(http.MethodPost, "/forms/"+f.ID+"/targets", strings.NewReader(`{"chapter_id":"chapter-2"}`)), "formID", f.ID)
	rec2 := httptest.NewRecorder()
	handler(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec2.Code, rec2.Body.String())
	}
}

func TestResolveLinkKey_NotFound(t *testing.T) {
	fm := newTestManager(t)
	handler := routes.ResolveLinkKey(fm)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/public-links/nope", nil), "key", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
