package routes_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestASendNamingAClosedPeriodIsRefusedThroughTheRouteTable(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	ctx := context.Background()
	weekly := models.FormSchedule{Start: "2026-10-05T09:00:00+03:00", Zone: "Africa/Nairobi", Rule: "FREQ=WEEKLY;INTERVAL=1"}
	f := seedForm(t, fm)
	f.Schedule = weekly
	if _, err := fm.Update(ctx, f); err != nil {
		t.Fatalf("Update: %v", err)
	}
	q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How was the week?"}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	prev, err := weekly.PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	cur, err := weekly.PeriodAt(time.Now())
	if err != nil {
		t.Fatalf("PeriodAt: %v", err)
	}
	body := `{"answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	if rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers?period_key="+prev.Key, body); rec.Code != http.StatusConflict {
		t.Fatalf("closed period: got %d, body %s", rec.Code, rec.Body.String())
	}
	rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers?period_key="+cur.Key, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("open period: got %d, body %s", rec.Code, rec.Body.String())
	}
	if got := decodeAnswers(t, rec); len(got) != 1 || got[0].PeriodKey != cur.Key {
		t.Fatalf("answers = %+v, want period %s", got, cur.Key)
	}
}
