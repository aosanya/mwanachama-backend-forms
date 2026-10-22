package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestF88AResidentReadsTheirOwnAnswersOnAFormThatNoLongerReachesThem(t *testing.T) {
	fm, sent, elsewhere, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)

	if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sent.form.ID+"/answers", sent.answerBody()); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("submit_answers = %d (%s)", rec.Code, rec.Body.String())
	}
	fm.UseGroups(groupsOf(map[string][]string{"alice": {}}))

	rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sent.form.ID+"/answers/mine", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), sent.question.ID) {
		t.Fatalf("answers/mine after leaving the group = %d %s, want 200 with what alice sent", rec.Code, rec.Body.String())
	}
	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sent.form.ID+"/answers/mine/previous", ""); rec.Code != http.StatusOK {
		t.Fatalf("answers/mine/previous after leaving the group = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+elsewhere.form.ID+"/answers/mine", ""); rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Fatalf("answers/mine on a form never sent to alice = %d %s, want 200 []", rec.Code, rec.Body.String())
	}
}

func TestF88AnOwnAnswersRefusalNamesNoGoMethod(t *testing.T) {
	fm, _, _, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)
	for _, path := range []string{"/forms/no-such-form/answers/mine", "/forms/no-such-form/answers/mine/previous"} {
		rec := doAs(t, mux, "alice", http.MethodGet, path, "")
		if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "Answers:") {
			t.Errorf("%s = %d %s, want 404 naming no method", path, rec.Code, rec.Body.String())
		}
	}
}
