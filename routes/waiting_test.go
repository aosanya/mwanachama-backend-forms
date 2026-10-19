package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

type waitingCallerKey struct{}

func waitingCallerFrom(ctx context.Context) string {
	id, _ := ctx.Value(waitingCallerKey{}).(string)
	return id
}

func mountAsCaller(t *testing.T, fm mwanachamaforms.FormManager) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(fm, routes.Mount{Caller: waitingCallerFrom}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		who := r.Header.Get("X-Test-Caller")
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), waitingCallerKey{}, who)))
	})
}

func waitingAs(t *testing.T, h http.Handler, caller, query string) ([]mwanachamaforms.WaitingForm, string) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/forms/waiting"+query, nil)
	r.Header.Set("X-Test-Caller", caller)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []mwanachamaforms.WaitingForm
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	return out, rec.Body.String()
}

func groupsOf(byCaller map[string][]string) mwanachamaforms.Groups {
	return func(_ context.Context, callerID string) ([]string, error) {
		return byCaller[callerID], nil
	}
}

func composeForm(t *testing.T, fm mwanachamaforms.FormManager, title, from string, targets []string, prompts int) mwanachamaforms.Form {
	t.Helper()
	ctx := context.Background()
	f, err := fm.Create(ctx, mwanachamaforms.Form{
		Title:               title,
		OriginatorChapterID: from,
		ClosesAt:            time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for i := 0; i < prompts; i++ {
		if _, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
			FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "Prompt",
		}}); err != nil {
			t.Fatalf("AddQuestion: %v", err)
		}
	}
	for _, c := range targets {
		if _, err := fm.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: c}); err != nil {
			t.Fatalf("AddTarget: %v", err)
		}
	}
	return f
}

func publish(t *testing.T, fm mwanachamaforms.FormManager, f mwanachamaforms.Form) mwanachamaforms.Form {
	t.Helper()
	out, _, err := fm.Publish(context.Background(), f.ID, "", "sender-1")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return out
}

func answerAll(t *testing.T, fm mwanachamaforms.FormManager, formID, memberID, chapterID string) {
	t.Helper()
	qs, err := fm.ListQuestions(context.Background(), formID)
	if err != nil {
		t.Fatalf("ListQuestions: %v", err)
	}
	answers := make([]mwanachamaforms.Answer, 0, len(qs))
	for _, q := range qs {
		answers = append(answers, mwanachamaforms.Answer{QuestionID: q.ID, ValueText: "yes"})
	}
	if _, err := fm.SubmitAnswers(context.Background(), formID, memberID, chapterID, answers); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
}

func byTitle(rows []mwanachamaforms.WaitingForm) map[string]mwanachamaforms.WaitingForm {
	out := map[string]mwanachamaforms.WaitingForm{}
	for _, r := range rows {
		out[r.Title] = r
	}
	return out
}

func TestWaitingListsOnlyOpenFormsThatReachedTheCallersGroups(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{
		"alice": {"ward-1", "ward-2"},
		"bob":   {"ward-9"},
	}))

	mine := publish(t, fm, composeForm(t, fm, "Mine", "council", []string{"ward-1"}, 3))
	both := publish(t, fm, composeForm(t, fm, "Both", "council", []string{"ward-1", "ward-2", "ward-9"}, 1))
	publish(t, fm, composeForm(t, fm, "Elsewhere", "council", []string{"ward-9"}, 1))
	composeForm(t, fm, "Unpublished", "council", []string{"ward-1"}, 1)
	closed := publish(t, fm, composeForm(t, fm, "Closed", "council", []string{"ward-2"}, 1))
	if _, err := fm.Close(context.Background(), closed.ID, "sender-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	rows, _ := waitingAs(t, mountAsCaller(t, fm), "alice", "")
	got := byTitle(rows)
	if len(rows) != 2 || got["Mine"].FormID != mine.ID || got["Both"].FormID != both.ID {
		t.Fatalf("alice sees %+v, want exactly Mine and Both", rows)
	}

	m := got["Mine"]
	if m.OriginatorChapterID != "council" || m.PublishedBy != "sender-1" || m.Questions != 3 ||
		m.ClosesAt != mine.ClosesAt || m.Answered {
		t.Fatalf("Mine row = %+v", m)
	}
	if strings.Join(got["Both"].ReachedVia, ",") != "ward-1,ward-2" {
		t.Fatalf("Both reached via %v, want only alice's own groups", got["Both"].ReachedVia)
	}
}

func TestWaitingTellsTheCallerWhetherTheyAnsweredAndNothingAboutAnyoneElse(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}, "bob": {"ward-1"}}))

	answered := publish(t, fm, composeForm(t, fm, "Answered", "council", []string{"ward-1"}, 2))
	publish(t, fm, composeForm(t, fm, "Pending", "council", []string{"ward-1"}, 2))
	othersOnly := publish(t, fm, composeForm(t, fm, "OthersOnly", "council", []string{"ward-1"}, 1))
	answerAll(t, fm, answered.ID, "alice", "ward-1")
	answerAll(t, fm, othersOnly.ID, "bob", "ward-1")
	for _, other := range []string{"bob", "carol", "dave"} {
		answerAll(t, fm, answered.ID, other, "ward-1")
	}

	h := mountAsCaller(t, fm)
	rows, raw := waitingAs(t, h, "alice", "")
	got := byTitle(rows)
	if !got["Answered"].Answered || got["Pending"].Answered || got["OthersOnly"].Answered {
		t.Fatalf("alice's answered flags wrong: %+v", rows)
	}
	for _, leak := range []string{"respondents", "by_status", "total", "member_id", "carol", "dave"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("waiting list leaks %q: %s", leak, raw)
		}
	}

	bobRows, _ := waitingAs(t, h, "bob", "")
	if !byTitle(bobRows)["Answered"].Answered {
		t.Fatalf("bob answered too: %+v", bobRows)
	}
	eveRows, _ := waitingAs(t, h, "eve", "")
	if len(eveRows) != 0 {
		t.Fatalf("a caller in no group sees %+v", eveRows)
	}
}

func TestWaitingTakesGroupsFromTheMountNeverTheRequest(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}}))
	publish(t, fm, composeForm(t, fm, "Theirs", "council", []string{"ward-9"}, 1))

	rows, _ := waitingAs(t, mountAsCaller(t, fm), "alice", "?chapter_id=ward-9&groups=ward-9&caller_id=bob")
	if len(rows) != 0 {
		t.Fatalf("request-named groups were honoured: %+v", rows)
	}
}

func TestWaitingWithNoResolverBelongsToNoGroups(t *testing.T) {
	fm := newTestManager(t)
	publish(t, fm, composeForm(t, fm, "Mine", "council", []string{"ward-1"}, 1))

	rows, raw := waitingAs(t, mountAsCaller(t, fm), "alice", "")
	if len(rows) != 0 || strings.TrimSpace(raw) != "[]" {
		t.Fatalf("a manager given no resolver answered %s", raw)
	}
}

func TestWaitingWithNoCallerAsksTheResolverNothing(t *testing.T) {
	fm := newTestManager(t)
	asked := false
	fm.UseGroups(func(context.Context, string) ([]string, error) {
		asked = true
		return []string{"ward-1"}, nil
	})
	publish(t, fm, composeForm(t, fm, "Mine", "council", []string{"ward-1"}, 1))

	rows, _ := waitingAs(t, mountAsCaller(t, fm), "", "")
	if len(rows) != 0 || asked {
		t.Fatalf("no caller: rows=%+v asked=%v", rows, asked)
	}
}

func TestWaitingSurfacesAResolverFailure(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(func(context.Context, string) ([]string, error) { return nil, errors.New("directory down") })

	r := httptest.NewRequest(http.MethodGet, "/forms/waiting", nil)
	r.Header.Set("X-Test-Caller", "alice")
	rec := httptest.NewRecorder()
	mountAsCaller(t, fm).ServeHTTP(rec, r)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "directory down") {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestWaitingDoesNotShadowReadingAFormByID(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	r := httptest.NewRequest(http.MethodGet, "/forms/"+f.ID, nil)
	rec := httptest.NewRecorder()
	mountAsCaller(t, fm).ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
