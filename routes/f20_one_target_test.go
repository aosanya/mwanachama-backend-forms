package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestASecondTargetOnAFormIsAccepted(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	mux := mount(t, fm)

	if rec := do(t, mux, http.MethodPost, "/forms/"+f.ID+"/targets", `{"chapter_id":"ward-7"}`); rec.Code != http.StatusCreated {
		t.Fatalf("first aim: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, mux, http.MethodPost, "/forms/"+f.ID+"/targets", `{"chapter_id":"ward-8"}`); rec.Code != http.StatusCreated {
		t.Fatalf("second aim: status = %d, body = %s, want 201", rec.Code, rec.Body.String())
	}
}

func aimedOpenForm(t *testing.T, fm mwanachamaforms.FormManager, group string) (mwanachamaforms.Form, mwanachamaforms.Question) {
	t.Helper()
	ctx := context.Background()
	f := seedForm(t, fm)
	q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is the ward?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if group != "" {
		if _, err := fm.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: group}); err != nil {
			t.Fatalf("AddTarget: %v", err)
		}
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f, q
}

func TestAnAnswerCountsTowardTheFormsOneTargetWhateverTheBodyNames(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := aimedOpenForm(t, fm, "ward-7")

	body := `{"chapter_id":"ward-elsewhere","answers":[{"question_id":"` + q.ID + `","value_text":"fine","chapter_id":"ward-elsewhere"}]}`
	rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit: got %d, body %s", rec.Code, rec.Body.String())
	}
	for _, a := range decodeAnswers(t, rec) {
		if a.ChapterID != "ward-7" {
			t.Fatalf("an answer counts toward %q; it must count toward the form's target, ward-7", a.ChapterID)
		}
	}
	held, err := fm.ListAnswers(context.Background(), f.ID, "member-caller")
	if err != nil {
		t.Fatalf("ListAnswers: %v", err)
	}
	if len(held) != 1 || held[0].ChapterID != "ward-7" {
		t.Fatalf("stored answers = %+v, want one counted toward ward-7", held)
	}
}

func TestAnAnswerOnAFormAimedNowhereCountsTowardNoGroup(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := aimedOpenForm(t, fm, "")

	body := `{"chapter_id":"ward-elsewhere","answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit: got %d, body %s", rec.Code, rec.Body.String())
	}
	for _, a := range decodeAnswers(t, rec) {
		if a.ChapterID != "" {
			t.Fatalf("an answer counts toward %q; a form aimed nowhere names no group", a.ChapterID)
		}
	}
}
