package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestFileIDsOnANonFileQuestionAreRefusedThroughTheRouteTable(t *testing.T) {
	fm := newTestManager(t)
	f, q := openFormWithOneQuestion(t, fm)
	mux := callerMount(t, fm)

	rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+f.ID+"/answers",
		`{"answers":[{"question_id":"`+q.ID+`","value_text":"fine","file_ids":["f1"]}]}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "file_ids") {
		t.Fatalf("free text answer carrying file_ids: %d %s, want 400 naming file_ids", rec.Code, rec.Body.String())
	}

	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+f.ID+"/answers/mine", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("answers/mine after the refusal = %d %s, want []", rec.Code, rec.Body.String())
	}

	rec = doAs(t, mux, "alice", http.MethodPost, "/forms/"+f.ID+"/answers",
		`{"answers":[{"question_id":"`+q.ID+`","value_text":"fine"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("the same answer without file_ids: %d %s, want 200", rec.Code, rec.Body.String())
	}
}
