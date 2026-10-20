package routes_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func mountReachedOnly(t *testing.T, fm mwanachamaforms.FormManager, reachedOnly bool) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(ctx context.Context) string {
		id, _ := ctx.Value(callerKey{}).(string)
		return id
	}
	for _, rt := range routes.RoutesFor(fm, routes.Mount{Caller: caller, ReachedOnly: reachedOnly}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return mux
}

type choiceForm struct {
	form     mwanachamaforms.Form
	question mwanachamaforms.Question
	option   mwanachamaforms.QuestionOption
}

func composeChoiceForm(t *testing.T, fm mwanachamaforms.FormManager, target string, publish bool) choiceForm {
	t.Helper()
	ctx := context.Background()
	f, err := fm.Create(ctx, mwanachamaforms.Form{
		Title:               "Water point",
		OriginatorChapterID: "county-1",
		ClosesAt:            time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, opts, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{
		Question: mwanachamaforms.Question{FormID: f.ID, AnswerType: mwanachamaforms.AnswerSingleChoice, Prompt: "Which tap?"},
		Options:  []mwanachamaforms.QuestionOption{{Label: "North"}, {Label: "South"}},
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := fm.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: target}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if publish {
		if _, _, err := fm.Publish(ctx, f.ID, "", "sender-1"); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}
	return choiceForm{form: f, question: q, option: opts[0]}
}

func (c choiceForm) answerBody() string {
	return `{"chapter_id":"ward-1","answers":[{"question_id":"` + c.question.ID + `","option_ids":["` + c.option.ID + `"]}]}`
}

func residentFixture(t *testing.T) (mwanachamaforms.FormManager, choiceForm, choiceForm, choiceForm) {
	t.Helper()
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}}))
	sent := composeChoiceForm(t, fm, "ward-1", true)
	elsewhere := composeChoiceForm(t, fm, "ward-9", true)
	unsent := composeChoiceForm(t, fm, "ward-1", false)
	return fm, sent, elsewhere, unsent
}

func TestF15AResidentReadsAndAnswersAFormSentToTheirGroup(t *testing.T) {
	fm, sent, _, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)

	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sent.form.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("get_form = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sent.form.ID+"/questions", "")
	if rec.Code != http.StatusOK || rec.Body.String() == "[]\n" {
		t.Fatalf("list_questions = %d %s, want the form's questions", rec.Code, rec.Body.String())
	}
	rec = doAs(t, mux, "alice", http.MethodGet, "/forms/"+sent.form.ID+"/questions/"+sent.question.ID+"/options", "")
	if rec.Code != http.StatusOK || rec.Body.String() == "[]\n" {
		t.Fatalf("list_options = %d %s, want the question's choices", rec.Code, rec.Body.String())
	}
	if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sent.form.ID+"/answers", sent.answerBody()); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("submit_answers = %d, want success (%s)", rec.Code, rec.Body.String())
	}
}

func TestF15AFormNotSentToTheResidentAnswersAsAnUnknownOne(t *testing.T) {
	fm, sent, elsewhere, unsent := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)
	const missing = "no-such-form"

	for _, hidden := range []struct {
		name string
		c    choiceForm
	}{{"a form sent to another group", elsewhere}, {"a form not yet sent", unsent}} {
		probes := []struct {
			what, method, path, missingPath, body string
		}{
			{"get_form", http.MethodGet, "/forms/" + hidden.c.form.ID, "/forms/" + missing, ""},
			{"list_questions", http.MethodGet, "/forms/" + hidden.c.form.ID + "/questions", "/forms/" + missing + "/questions", ""},
			{"list_options", http.MethodGet, "/forms/" + hidden.c.form.ID + "/questions/" + hidden.c.question.ID + "/options", "/forms/" + missing + "/questions/no-such-question/options", ""},
			{"list_options through a sent form's address", http.MethodGet, "/forms/" + sent.form.ID + "/questions/" + hidden.c.question.ID + "/options", "/forms/" + sent.form.ID + "/questions/no-such-question/options", ""},
			{"submit_answers", http.MethodPost, "/forms/" + hidden.c.form.ID + "/answers", "/forms/" + missing + "/answers", hidden.c.answerBody()},
		}
		for _, p := range probes {
			got := doAs(t, mux, "alice", p.method, p.path, p.body)
			want := doAs(t, mux, "alice", p.method, p.missingPath, p.body)
			if got.Code != want.Code || got.Body.String() != want.Body.String() {
				t.Errorf("%s, %s = %d %q; an unknown one answers %d %q",
					hidden.name, p.what, got.Code, got.Body.String(), want.Code, want.Body.String())
			}
		}
		answers, err := fm.ListAllAnswers(context.Background(), hidden.c.form.ID, "")
		if err != nil {
			t.Fatalf("ListAllAnswers: %v", err)
		}
		if len(answers) != 0 {
			t.Errorf("%s took %d answers from a resident it never reached", hidden.name, len(answers))
		}
	}
}

func TestF15AResidentWithNoGroupsOrNoCallerReachesNothing(t *testing.T) {
	fm, sent, _, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)
	for _, who := range []string{"", "bob"} {
		if rec := doAs(t, mux, who, http.MethodGet, "/forms/"+sent.form.ID, ""); rec.Code != http.StatusNotFound {
			t.Errorf("caller %q get_form = %d, want 404", who, rec.Code)
		}
	}
}

func TestF15AnOperatorMountIsNotNarrowed(t *testing.T) {
	fm, _, elsewhere, unsent := residentFixture(t)
	mux := mountReachedOnly(t, fm, false)
	for _, c := range []choiceForm{elsewhere, unsent} {
		if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+c.form.ID, ""); rec.Code != http.StatusOK {
			t.Errorf("operator get_form = %d, want 200", rec.Code)
		}
	}
}
