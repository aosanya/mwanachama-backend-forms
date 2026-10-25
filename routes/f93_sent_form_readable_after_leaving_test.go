package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestF93ASentFormStaysReadableButNotAnswerableAfterLeavingItsGroup(t *testing.T) {
	fm, sent, _, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)

	if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sent.form.ID+"/answers", sent.answerBody()); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("submit_answers = %d (%s)", rec.Code, rec.Body.String())
	}
	fm.UseGroups(groupsOf(map[string][]string{"alice": {}}))

	reads := []struct{ name, path, want string }{
		{"get_form", "/forms/" + sent.form.ID, sent.form.ID},
		{"list_questions", "/forms/" + sent.form.ID + "/questions", sent.question.ID},
		{"list_options", "/forms/" + sent.form.ID + "/questions/" + sent.question.ID + "/options", sent.option.ID},
		{"answers/mine", "/forms/" + sent.form.ID + "/answers/mine", sent.question.ID},
		{"submissions/mine", "/forms/" + sent.form.ID + "/submissions/mine", sent.form.ID},
	}
	for _, r := range reads {
		rec := doAs(t, mux, "alice", http.MethodGet, r.path, "")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), r.want) {
			t.Errorf("%s after leaving the group = %d %s, want 200 naming %s", r.name, rec.Code, rec.Body.String(), r.want)
		}
	}

	if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sent.form.ID+"/answers", sent.answerBody()); rec.Code != http.StatusNotFound {
		t.Fatalf("submit_answers after leaving the group = %d %s, want 404", rec.Code, rec.Body.String())
	}
}

func TestF93AFormNeverSentStaysHiddenAfterLeavingItsGroup(t *testing.T) {
	fm, sent, _, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {}}))

	for _, path := range []string{"/forms/" + sent.form.ID, "/forms/" + sent.form.ID + "/submissions/mine"} {
		if rec := doAs(t, mux, "alice", http.MethodGet, path, ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s with nothing sent = %d %s, want 404", path, rec.Code, rec.Body.String())
		}
	}
	if rec := doAs(t, mux, "bob", http.MethodGet, "/forms/"+sent.form.ID, ""); rec.Code != http.StatusNotFound {
		t.Errorf("get_form by a caller who never sent it = %d, want 404", rec.Code)
	}
}
