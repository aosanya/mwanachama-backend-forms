package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func callAs(t *testing.T, h http.Handler, method, path, caller, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("X-Test-Caller", caller)
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func reportOf(t *testing.T, rec *httptest.ResponseRecorder) mwanachamaforms.ResultReport {
	t.Helper()
	var out mwanachamaforms.ResultReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	return out
}

func TestPublishingResultsRecordsVersionsAndAResidentReadsTheLatest(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}, "bob": {"ward-2"}}))
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"ward-1"}, 1))
	h := mountAsCaller(t, fm)
	path := "/forms/" + f.ID + "/results"

	if rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc"}`); rec.Code != http.StatusConflict {
		t.Fatalf("publishing an open form: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if rec := callAs(t, h, http.MethodGet, path+"/latest", "alice", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("before any report: status = %d", rec.Code)
	}
	if rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("no commit: status = %d", rec.Code)
	}

	rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc","published_by":"mallory"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	first := reportOf(t, rec)
	if first.Version != 1 || first.ChapterID != "ward-1" || first.PublishedBy != "clerk" || first.PublishedAt == "" {
		t.Fatalf("first = %+v", first)
	}
	second := reportOf(t, callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"def"}`))
	if second.Version != 2 {
		t.Fatalf("second = %+v", second)
	}

	got, err := fm.Get(context.Background(), f.ID)
	if err != nil || got.ResultsPublishedAt != second.PublishedAt {
		t.Fatalf("results_published_at = %q, err = %v", got.ResultsPublishedAt, err)
	}

	rec = callAs(t, h, http.MethodGet, path+"/latest", "alice", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("alice reads: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if latest := reportOf(t, rec); latest.Version != 2 || latest.GitCommit != "def" || latest.GitPath != "r/water.md" {
		t.Fatalf("latest = %+v", latest)
	}

	outsider := callAs(t, h, http.MethodGet, path+"/latest", "bob", "")
	missing := callAs(t, h, http.MethodGet, "/forms/no-such-form/results/latest", "alice", "")
	if outsider.Code != http.StatusNotFound || missing.Code != http.StatusNotFound || outsider.Body.String() != missing.Body.String() {
		t.Fatalf("outsider %d %q, missing %d %q", outsider.Code, outsider.Body.String(), missing.Code, missing.Body.String())
	}
}

func TestAReportOnAFormAimedAtSeveralGroupsIsReadInEveryOne(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}, "bob": {"ward-2"}, "carol": {"ward-3"}}))
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"ward-1", "ward-2"}, 1))
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	h := mountAsCaller(t, fm)
	path := "/forms/" + f.ID + "/results"
	if rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc"}`); rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	for _, reader := range []string{"alice", "bob"} {
		rec := callAs(t, h, http.MethodGet, path+"/latest", reader, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s reads: status = %d, body = %s", reader, rec.Code, rec.Body.String())
		}
		if got := reportOf(t, rec); got.GitCommit != "abc" {
			t.Fatalf("%s reads %+v", reader, got)
		}
	}

	outsider := callAs(t, h, http.MethodGet, path+"/latest", "carol", "")
	missing := callAs(t, h, http.MethodGet, "/forms/no-such-form/results/latest", "carol", "")
	if outsider.Code != http.StatusNotFound || outsider.Body.String() != missing.Body.String() {
		t.Fatalf("outsider %d %q, missing %d %q", outsider.Code, outsider.Body.String(), missing.Code, missing.Body.String())
	}
}
