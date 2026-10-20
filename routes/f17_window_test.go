package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestComposingAFormOverTheWireKeepsItsWindowUnlessTheBodySaysAdvisory(t *testing.T) {
	fm := newTestManager(t)
	mux := mount(t, fm)
	closes := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)

	for body, want := range map[string]bool{
		`{"title":"Kept","closes_at":"` + closes + `"}`:                            false,
		`{"title":"Advisory","closes_at":"` + closes + `","window_advisory":true}`: true,
	} {
		rec := do(t, mux, http.MethodPost, "/forms", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
		}
		var f mwanachamaforms.Form
		if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if f.WindowAdvisory != want {
			t.Fatalf("%s: window_advisory = %v, want %v", body, f.WindowAdvisory, want)
		}
	}
}

func TestAnAnswerBeforeAKeptWindowOpensIsAConflict(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	ctx := context.Background()
	f, err := fm.Create(ctx, mwanachamaforms.Form{
		Title:    "Early",
		OpensAt:  time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		ClosesAt: time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is the ward?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	body := `{"answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("submit before opens_at: got %d, body %s", rec.Code, rec.Body.String())
	}
}
