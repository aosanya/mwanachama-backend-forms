package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

type callerKey struct{}

func callerMount(t *testing.T, fm mwanachamaforms.FormManager) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(ctx context.Context) string {
		id, _ := ctx.Value(callerKey{}).(string)
		return id
	}
	for _, rt := range routes.RoutesFor(fm, routes.Mount{Caller: caller}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return mux
}

func doAs(t *testing.T, mux *http.ServeMux, caller, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	if caller != "" {
		r = r.WithContext(context.WithValue(r.Context(), callerKey{}, caller))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func openFormWithOneQuestion(t *testing.T, fm mwanachamaforms.FormManager) (mwanachamaforms.Form, mwanachamaforms.Question) {
	t.Helper()
	f := seedForm(t, fm)
	q, _, err := fm.AddQuestion(context.Background(), mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "How is the ward?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := fm.Publish(context.Background(), f.ID, "", "author-1"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f, q
}

func decodeAnswers(t *testing.T, rec *httptest.ResponseRecorder) []mwanachamaforms.Answer {
	t.Helper()
	var out []mwanachamaforms.Answer
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode answers: %v (body %s)", err, rec.Body.String())
	}
	return out
}

func TestSubmittingAnswersFilesThemUnderTheCallerWhateverTheBodyNames(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := openFormWithOneQuestion(t, fm)

	body := `{"member_id":"member-victim","chapter_id":"chapter-1","answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	rec := doAs(t, mux, "member-caller", http.MethodPost, "/forms/"+f.ID+"/answers", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit: got %d, body %s", rec.Code, rec.Body.String())
	}
	for _, a := range decodeAnswers(t, rec) {
		if a.MemberID != "member-caller" {
			t.Fatalf("an answer was filed under %q; it must be the caller's own, member-caller", a.MemberID)
		}
	}

	held, err := fm.ListAnswers(context.Background(), f.ID, "member-victim")
	if err != nil {
		t.Fatalf("ListAnswers: %v", err)
	}
	if len(held) != 0 {
		t.Fatalf("the member the body named holds %d answers it never gave: %+v", len(held), held)
	}
}

func TestSubmittingAnswersWithNoCallerIsRefused(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := openFormWithOneQuestion(t, fm)

	body := `{"member_id":"member-victim","answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	rec := doAs(t, mux, "", http.MethodPost, "/forms/"+f.ID+"/answers", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("submit with no caller: got %d, body %s, want 400", rec.Code, rec.Body.String())
	}
}

func TestReadingAnswersReturnsOnlyTheCallersOwn(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := openFormWithOneQuestion(t, fm)

	for _, member := range []string{"member-caller", "member-other"} {
		if _, err := fm.SubmitAnswers(context.Background(), f.ID, member, []mwanachamaforms.Answer{
			{QuestionID: q.ID, ValueText: "from " + member},
		}); err != nil {
			t.Fatalf("SubmitAnswers %s: %v", member, err)
		}
	}

	rec := doAs(t, mux, "member-caller", http.MethodGet, "/forms/"+f.ID+"/answers/mine", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("read own answers: got %d, body %s", rec.Code, rec.Body.String())
	}
	got := decodeAnswers(t, rec)
	if len(got) != 1 || got[0].MemberID != "member-caller" || got[0].ValueText != "from member-caller" {
		t.Fatalf("own answers = %+v, want exactly member-caller's one answer", got)
	}

	rec = doAs(t, mux, "member-caller", http.MethodGet, "/forms/"+f.ID+"/answers/member-other", "")
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "from member-other") {
		t.Fatalf("naming another member in the path still reads their answers: %s", rec.Body.String())
	}
}

func TestEveryonesAnswersAreReadUnderTheirOwnAction(t *testing.T) {
	fm := newTestManager(t)
	mux := callerMount(t, fm)
	f, q := openFormWithOneQuestion(t, fm)

	for _, member := range []string{"member-a", "member-b"} {
		if _, err := fm.SubmitAnswers(context.Background(), f.ID, member, []mwanachamaforms.Answer{
			{QuestionID: q.ID, ValueText: "from " + member},
		}); err != nil {
			t.Fatalf("SubmitAnswers %s: %v", member, err)
		}
	}

	actions := map[string]string{}
	for _, rt := range routes.Routes(fm) {
		actions[rt.Method+" "+rt.Path] = rt.Action
	}
	if got := actions["GET /forms/{form_id}/answers"]; got != "forms.answer.list_all" {
		t.Fatalf("everyone's answers are gated by %q, want forms.answer.list_all", got)
	}
	if got := actions["GET /forms/{form_id}/answers/mine"]; got != "forms.answer.list" {
		t.Fatalf("the caller's own answers are gated by %q, want forms.answer.list", got)
	}

	rec := doAs(t, mux, "operator-1", http.MethodGet, "/forms/"+f.ID+"/answers", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("read everyone's answers: got %d, body %s", rec.Code, rec.Body.String())
	}
	if got := decodeAnswers(t, rec); len(got) != 2 {
		t.Fatalf("everyone's answers = %+v, want both members'", got)
	}

	rec = doAs(t, mux, "operator-1", http.MethodGet, "/forms/"+f.ID+"/answers?member_id=member-b", "")
	got := decodeAnswers(t, rec)
	if rec.Code != http.StatusOK || len(got) != 1 || got[0].MemberID != "member-b" {
		t.Fatalf("everyone's answers narrowed to member-b: got %d %+v", rec.Code, got)
	}
}
