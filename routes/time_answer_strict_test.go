package routes_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestATimeAnswerIsStoredOnlyAsTwoDigitHoursAndMinutes(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	q, _, err := fm.AddQuestion(context.Background(), mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: models.AnswerTime, Prompt: "When does the bus leave?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := fm.Publish(context.Background(), f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	mux := callerMount(t, fm)

	rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+f.ID+"/answers",
		`{"answers":[{"question_id":"`+q.ID+`","value_time":"7:05"}]}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "HH:MM") {
		t.Fatalf("value_time 7:05: %d %s, want 400 naming HH:MM", rec.Code, rec.Body.String())
	}
	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+f.ID+"/answers/mine", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("answers/mine after the refusal = %d %s, want []", rec.Code, rec.Body.String())
	}

	rec = doAs(t, mux, "alice", http.MethodPost, "/forms/"+f.ID+"/answers",
		`{"answers":[{"question_id":"`+q.ID+`","value_time":"07:05"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("value_time 07:05: %d %s, want 200", rec.Code, rec.Body.String())
	}
	stored := decodeAnswers(t, doAs(t, mux, "alice", http.MethodGet, "/forms/"+f.ID+"/answers/mine", ""))
	if len(stored) != 1 || stored[0].ValueTime != "07:05" {
		t.Fatalf("stored answers %+v, want one value_time 07:05", stored)
	}
}
