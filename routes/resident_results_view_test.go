package routes_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestAResidentReadingTheLatestResultsReadsNoStaffMemberId(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}}))
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"ward-1"}, 1))
	h := mountAsCaller(t, fm)
	path := "/forms/" + f.ID + "/results"
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc"}`); rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec := doAs(t, mountReachedOnly(t, fm, true), "alice", http.MethodGet, path+"/latest", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("alice reads: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "clerk") {
		t.Fatalf("a resident reads the publisher's member id: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "r/water.md") {
		t.Fatalf("the resident view lost the report path: %s", rec.Body.String())
	}
}
