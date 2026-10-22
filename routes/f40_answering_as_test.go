package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestF40ASendNamesTheGroupItAnswersAs(t *testing.T) {
	fm := newTestManager(t)
	ctx := context.Background()
	f := seedForm(t, fm)
	q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is trade?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	for _, g := range []string{"traders", "teachers"} {
		if _, err := fm.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: g}); err != nil {
			t.Fatalf("AddTarget %s: %v", g, err)
		}
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	fm.UseGroups(func(context.Context, string) ([]string, error) { return []string{"traders", "teachers"}, nil })
	mux := callerMount(t, fm)

	body := `{"answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	for _, g := range []string{"teachers", "traders"} {
		if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+f.ID+"/answers?as="+g, body); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("answer as %s: %d %s", g, rec.Code, rec.Body)
		}
	}
	if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+f.ID+"/answers?as=farmers", body); rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("answer as a group the form was not sent to: %d %s, want a 4xx", rec.Code, rec.Body)
	}
	sends, err := fm.ListSubmissions(ctx, f.ID, "alice", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(sends) != 2 {
		t.Fatalf("sends = %+v, want one per group answered as", sends)
	}
}
