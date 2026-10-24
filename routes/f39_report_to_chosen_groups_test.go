package routes_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
)

func TestAReportReachesTheGroupsChosenAtPublishAndTheirInnerGroups(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{
		"alice": {"ward-1"},
		"erin":  {"finance"},
		"dave":  {"finance-desk", "finance"},
		"bob":   {"ward-2"},
	}))
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"ward-1", "ward-2"}, 1))
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	h := mountAsCaller(t, fm)
	path := "/forms/" + f.ID + "/results"

	rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc","chapter_ids":["ward-1","finance","ward-1"," "]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := reportOf(t, rec); got.ChapterID != "ward-1" || !slices.Equal(got.ChapterIDs, []string{"ward-1", "finance"}) {
		t.Fatalf("report = %+v", got)
	}

	for _, reader := range []string{"alice", "erin", "dave"} {
		if rec := callAs(t, h, http.MethodGet, path+"/latest", reader, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s reads: status = %d, body = %s", reader, rec.Code, rec.Body.String())
		}
	}

	outsider := callAs(t, h, http.MethodGet, path+"/latest", "bob", "")
	missing := callAs(t, h, http.MethodGet, "/forms/no-such-form/results/latest", "bob", "")
	if outsider.Code != http.StatusNotFound || outsider.Body.String() != missing.Body.String() {
		t.Fatalf("a target left unchosen: %d %q, missing %d %q", outsider.Code, outsider.Body.String(), missing.Code, missing.Body.String())
	}
}

func TestAReportWithNoGroupsChosenGoesToEveryTarget(t *testing.T) {
	fm := newTestManager(t)
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"ward-1", "ward-2"}, 1))
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	h := mountAsCaller(t, fm)
	rec := callAs(t, h, http.MethodPost, "/forms/"+f.ID+"/results", "clerk", `{"git_path":"r/water.md","git_commit":"abc"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := reportOf(t, rec); !slices.Equal(got.ChapterIDs, []string{"ward-1", "ward-2"}) {
		t.Fatalf("report = %+v", got)
	}
}

func TestAReportReachesAnInnerGroupTheHostNamesOnlyAsMembersWithin(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{
		"irene": {"inner-ward"},
		"bob":   {"ward-2"},
	}))
	fm.UseMembers(func(_ context.Context, groupID string, includesDescendants bool) ([]string, error) {
		if groupID == "district" && includesDescendants {
			return []string{"irene"}, nil
		}
		return nil, nil
	})
	f := publish(t, fm, composeForm(t, fm, "Water", "council", []string{"district", "ward-2"}, 1))
	if _, err := fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	h := mountAsCaller(t, fm)
	path := "/forms/" + f.ID + "/results"
	if rec := callAs(t, h, http.MethodPost, path, "clerk", `{"git_path":"r/water.md","git_commit":"abc","chapter_ids":["district"]}`); rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := callAs(t, h, http.MethodGet, path+"/latest", "irene", ""); rec.Code != http.StatusOK {
		t.Fatalf("inner-group resident reads: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := callAs(t, h, http.MethodGet, path+"/latest", "bob", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unchosen target reads: status = %d", rec.Code)
	}
}
