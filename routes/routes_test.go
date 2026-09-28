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
	s, err := mwanachamaforms.SpecFor("routestest")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	fm, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	return fm
}

func mount(t *testing.T, fm mwanachamaforms.FormManager) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.Routes(fm) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return mux
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func seedForm(t *testing.T, fm mwanachamaforms.FormManager) mwanachamaforms.Form {
	t.Helper()
	f, err := fm.Create(context.Background(), mwanachamaforms.Form{
		Title:               "Seed",
		OriginatorChapterID: "chapter-1",
		ClosesAt:            time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	return f
}

func TestCreateForm(t *testing.T) {
	mux := mount(t, newTestManager(t))

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	body := `{"title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"` + closesAt + `"}`
	rec := do(t, mux, http.MethodPost, "/forms", body)
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
	mux := mount(t, newTestManager(t))
	rec := do(t, mux, http.MethodPost, "/forms", `{"closes_at":"2099-01-01T00:00:00Z"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetForm_NotFound(t *testing.T) {
	mux := mount(t, newTestManager(t))
	rec := do(t, mux, http.MethodGet, "/forms/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPublishForm(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	rec := do(t, mount(t, fm), http.MethodPost, "/forms/"+f.ID+"/publish", `{"published_by":"admin"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var out struct {
		Form mwanachamaforms.Form `json:"form"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Form.Status != mwanachamaforms.StatusOpen {
		t.Fatalf("status = %q, want open", out.Form.Status)
	}
}

func TestAddTarget_DuplicateConflict(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	mux := mount(t, fm)

	if rec := do(t, mux, http.MethodPost, "/forms/"+f.ID+"/targets", `{"chapter_id":"chapter-2"}`); rec.Code != http.StatusCreated {
		t.Fatalf("first add: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, mux, http.MethodPost, "/forms/"+f.ID+"/targets", `{"chapter_id":"chapter-2"}`); rec.Code != http.StatusConflict {
		t.Fatalf("second add: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestAddTargetIgnoresAFormIDInTheBody(t *testing.T) {
	fm := newTestManager(t)
	addressed, other := seedForm(t, fm), seedForm(t, fm)
	mux := mount(t, fm)

	body := `{"chapter_id":"chapter-2","form_id":"` + other.ID + `"}`
	rec := do(t, mux, http.MethodPost, "/forms/"+addressed.ID+"/targets", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var out mwanachamaforms.Target
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.FormID != addressed.ID {
		t.Fatalf("form_id = %q, want the form in the address (%q)", out.FormID, addressed.ID)
	}
}

func TestAddQuestionTakesItsFormFromTheAddress(t *testing.T) {
	fm := newTestManager(t)
	addressed, other := seedForm(t, fm), seedForm(t, fm)
	mux := mount(t, fm)

	body := `{"answer_type":"free_text","prompt":"Why?","form_id":"` + other.ID + `"}`
	rec := do(t, mux, http.MethodPost, "/forms/"+addressed.ID+"/questions", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var out struct {
		Question mwanachamaforms.Question `json:"question"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Question.FormID != addressed.ID {
		t.Fatalf("form_id = %q, want the form in the address (%q)", out.Question.FormID, addressed.ID)
	}
}

func TestOpenPublicLink_NotFound(t *testing.T) {
	mux := mount(t, newTestManager(t))
	rec := do(t, mux, http.MethodGet, "/public-links/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"error":"not found"}` {
		t.Errorf("body = %s, want a refusal that names no reason", body)
	}
}

func TestDeleteFormAnswersNoContent(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	rec := do(t, mount(t, fm), http.MethodDelete, "/forms/"+f.ID, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %s, want none", rec.Body.String())
	}
}

func TestRegisterFormsReadsItsFilterFromTheQuery(t *testing.T) {
	fm := newTestManager(t)
	seedForm(t, fm)
	rec := do(t, mount(t, fm), http.MethodGet, "/forms?q=seed&chapter_id=chapter-1&limit=10&offset=0", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var page mwanachamaforms.RegisterPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if page.Total != 1 || len(page.Rows) != 1 {
		t.Fatalf("page = %+v, want the one seeded form", page)
	}
}
