package routes_test

import (
	"context"
	"net/http"
	"testing"
)

func TestF41SubmittingAnswersRecordsTheLanguageTheQueryNames(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := openFormWithOneQuestion(t, fm)

	body := `{"language":"fr","answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers?language=sw", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit: got %d, body %s", rec.Code, rec.Body.String())
	}
	sent, err := fm.GetSubmission(context.Background(), f.ID, "member-caller")
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if sent.Language != "sw" {
		t.Fatalf("language = %q, want sw from the query", sent.Language)
	}
}
