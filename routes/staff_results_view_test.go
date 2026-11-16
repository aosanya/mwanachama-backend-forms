package routes_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestStaffReadingTheLatestResultsKeepsThePublishersMemberId(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}, "clerk": {"ward-1"}}))
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"ward-1"}, 1))
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	path := "/forms/" + f.ID + "/results"
	if rec := callAs(t, mountAsCaller(t, fm), http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc"}`); rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	staff := doAs(t, mountReachedOnly(t, fm, false), "clerk", http.MethodGet, path+"/latest", "")
	if staff.Code != http.StatusOK || !strings.Contains(staff.Body.String(), `"published_by":"clerk"`) {
		t.Fatalf("staff read = %d %s, want the whole report with published_by", staff.Code, staff.Body.String())
	}

	resident := doAs(t, mountReachedOnly(t, fm, true), "alice", http.MethodGet, path+"/latest", "")
	if resident.Code != http.StatusOK || strings.Contains(resident.Body.String(), "clerk") {
		t.Fatalf("resident read = %d %s, want the respondent view", resident.Code, resident.Body.String())
	}
}
