package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

type listWorld struct {
	fm mwanachamaforms.FormManager
	h  http.Handler
}

func newListWorld(t *testing.T, groups mwanachamaforms.Groups) *listWorld {
	t.Helper()
	fm := newTestManager(t)
	if groups != nil {
		fm.UseGroups(groups)
	}
	return &listWorld{fm: fm, h: mountAsCaller(t, fm)}
}

func (w *listWorld) closedForm(t *testing.T, title string, targets ...string) mwanachamaforms.Form {
	t.Helper()
	f := publish(t, w.fm, composeForm(t, w.fm, title, "council", targets, 1))
	if _, err := w.fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return f
}

func (w *listWorld) publishReport(t *testing.T, formID, commit string, groups ...string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"git_path": "reports/" + formID + ".md", "git_commit": commit, "chapter_ids": groups})
	if rec := callAs(t, w.h, http.MethodPost, "/forms/"+formID+"/results", "clerk", string(body)); rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	time.Sleep(3 * time.Millisecond)
}

func (w *listWorld) list(t *testing.T, caller string) []mwanachamaforms.PublishedReport {
	t.Helper()
	rec := callAs(t, w.h, http.MethodGet, "/forms/reports/mine", caller, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s lists: status = %d, body = %s", caller, rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) == "null" {
		t.Fatalf("%s lists null, not an empty list", caller)
	}
	var out []mwanachamaforms.PublishedReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	return out
}

func listedForms(reports []mwanachamaforms.PublishedReport) []string {
	ids := make([]string, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.FormID)
	}
	return ids
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestReportsAreListedNewestPublishedFirstWhateverTheFormIds(t *testing.T) {
	w := newListWorld(t, groupsOf(map[string][]string{"alice": {"ward-1"}}))
	first := w.closedForm(t, "Roads", "ward-1")
	second := w.closedForm(t, "Water", "ward-1")

	w.publishReport(t, first.ID, "a1", "ward-1")
	w.publishReport(t, second.ID, "b1", "ward-1")
	if got := listedForms(w.list(t, "alice")); !equalIDs(got, []string{second.ID, first.ID}) {
		t.Fatalf("listed %v, want the form published last first: %v", got, []string{second.ID, first.ID})
	}

	w.publishReport(t, first.ID, "a2", "ward-1")
	got := w.list(t, "alice")
	if ids := listedForms(got); !equalIDs(ids, []string{first.ID, second.ID}) || got[0].Version != 2 {
		t.Fatalf("after a second version of the first form, listed %+v", got)
	}
}

func TestAResidentInAGroupInsideAPublishedGroupListsTheReport(t *testing.T) {
	w := newListWorld(t, groupsOf(map[string][]string{"alice": {"ward-1"}, "irene": {"inner-ward"}}))
	w.fm.UseMembers(func(_ context.Context, groupID string, includesDescendants bool) ([]string, error) {
		if groupID == "district" && includesDescendants {
			return []string{"irene"}, nil
		}
		return nil, nil
	})
	f := w.closedForm(t, "Water", "district")
	w.publishReport(t, f.ID, "abc", "district")

	if got := w.list(t, "irene"); len(got) != 1 || got[0].FormID != f.ID {
		t.Fatalf("irene lists %+v", got)
	}
	if got := w.list(t, "alice"); len(got) != 0 {
		t.Fatalf("alice, who is not inside the district, lists %+v", got)
	}
}

func TestAReportPublishedToTwoGroupsIsListedOnceForAMemberOfEither(t *testing.T) {
	w := newListWorld(t, groupsOf(map[string][]string{
		"alice": {"ward-1"},
		"bob":   {"ward-2"},
		"carol": {"ward-1", "ward-2"},
		"dave":  {"ward-3"},
	}))
	f := w.closedForm(t, "Annual report", "ward-1", "ward-2", "ward-3")
	w.publishReport(t, f.ID, "abc", "ward-1", "ward-2")

	for _, caller := range []string{"alice", "bob", "carol"} {
		if got := w.list(t, caller); len(got) != 1 || got[0].FormID != f.ID || got[0].Version != 1 {
			t.Fatalf("%s lists %+v", caller, got)
		}
	}
	if got := w.list(t, "dave"); len(got) != 0 {
		t.Fatalf("dave lists %+v", got)
	}
}

func TestAFailingGroupLookupIsAnErrorNotAnEmptyList(t *testing.T) {
	w := newListWorld(t, func(context.Context, string) ([]string, error) {
		return nil, errors.New("the groups service is down")
	})
	f := w.closedForm(t, "Annual report", "ward-1")
	w.publishReport(t, f.ID, "abc", "ward-1")

	rec := callAs(t, w.h, http.MethodGet, "/forms/reports/mine", "alice", "")
	if rec.Code == http.StatusOK {
		t.Fatalf("a failing group lookup answered 200 with %s", rec.Body.String())
	}
}

func TestAManagerWithoutAGroupLookupListsAnEmptyListNotNull(t *testing.T) {
	w := newListWorld(t, nil)
	f := w.closedForm(t, "Annual report", "ward-1")
	w.publishReport(t, f.ID, "abc", "ward-1")

	if got := w.list(t, "alice"); len(got) != 0 {
		t.Fatalf("alice lists %+v", got)
	}
}

func TestAMemberOfTheSameGroupInAnotherInstanceListsNothing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	build := func(instance string) *listWorld {
		spec, err := mwanachamaforms.SpecFor(instance)
		if err != nil {
			t.Fatalf("SpecFor: %v", err)
		}
		if err := mwanachamaforms.Provision(db, spec); err != nil {
			t.Fatalf("Provision: %v", err)
		}
		fm, err := mwanachamaforms.NewFormManager(db, spec)
		if err != nil {
			t.Fatalf("NewFormManager: %v", err)
		}
		fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}}))
		return &listWorld{fm: fm, h: mountAsCaller(t, fm)}
	}
	here := build("here1")
	elsewhere := build("else1")

	f := here.closedForm(t, "Annual report", "ward-1")
	here.publishReport(t, f.ID, "abc", "ward-1")

	if got := here.list(t, "alice"); len(got) != 1 {
		t.Fatalf("alice lists %+v in the instance that published", got)
	}
	if got := elsewhere.list(t, "alice"); len(got) != 0 {
		t.Fatalf("alice lists %+v in another instance", got)
	}
}
