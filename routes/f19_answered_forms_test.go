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

func answeredAs(t *testing.T, h http.Handler, caller, query string) ([]mwanachamaforms.AnsweredForm, string) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/forms/answered"+query, nil)
	r.Header.Set("X-Test-Caller", caller)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []mwanachamaforms.AnsweredForm
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	return out, rec.Body.String()
}

func titlesOf(rows []mwanachamaforms.AnsweredForm) string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Title)
	}
	return strings.Join(out, ",")
}

func TestAnsweredListsEveryFormTheCallerAnsweredOpenOrClosedNewestSentFirst(t *testing.T) {
	fm := newTestManager(t)
	oldest := publish(t, fm, composeForm(t, fm, "Oldest", "council", []string{"ward-1"}, 1))
	middle := publish(t, fm, composeForm(t, fm, "Middle", "council", []string{"ward-1"}, 1))
	newest := publish(t, fm, composeForm(t, fm, "Newest", "council", []string{"ward-1"}, 2))
	publish(t, fm, composeForm(t, fm, "Unanswered", "council", []string{"ward-1"}, 1))

	answerAll(t, fm, oldest.ID, "alice")
	answerAll(t, fm, newest.ID, "alice")
	answerAll(t, fm, middle.ID, "bob")
	closed, err := fm.Close(context.Background(), oldest.ID, "sender-1")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	rows, _ := answeredAs(t, mountAsCaller(t, fm), "alice", "")
	if titlesOf(rows) != "Newest,Oldest" {
		t.Fatalf("alice sees %s, want Newest,Oldest", titlesOf(rows))
	}
	n, o := rows[0], rows[1]
	if !n.Open || n.Status != mwanachamaforms.StatusOpen || n.PublishedAt != newest.PublishedAt ||
		n.OriginatorChapterID != "council" || n.PublishedBy != "sender-1" || n.ClosesAt != newest.ClosesAt ||
		n.AnsweredAt == "" || n.EditedAt != "" {
		t.Fatalf("Newest row = %+v", n)
	}
	if o.Open || o.Status != mwanachamaforms.StatusClosed || o.ClosedAt != closed.ClosedAt ||
		o.PublishedAt != oldest.PublishedAt {
		t.Fatalf("Oldest row = %+v", o)
	}
}

func TestAnsweredReadsOnlyTheCallersOwnAnswersWhateverTheRequestNames(t *testing.T) {
	fm := newTestManager(t)
	theirs := publish(t, fm, composeForm(t, fm, "Theirs", "council", []string{"ward-1"}, 1))
	for _, other := range []string{"bob", "carol"} {
		answerAll(t, fm, theirs.ID, other)
	}

	h := mountAsCaller(t, fm)
	rows, raw := answeredAs(t, h, "alice", "?member_id=bob&caller_id=bob")
	if len(rows) != 0 || strings.TrimSpace(raw) != "[]" {
		t.Fatalf("a request-named member was honoured: %s", raw)
	}

	bobRows, bobRaw := answeredAs(t, h, "bob", "")
	if titlesOf(bobRows) != "Theirs" {
		t.Fatalf("bob sees %s", titlesOf(bobRows))
	}
	for _, leak := range []string{"carol", "member_id", "respondents"} {
		if strings.Contains(bobRaw, leak) {
			t.Fatalf("answered list leaks %q: %s", leak, bobRaw)
		}
	}
}

func TestAnsweredWithNoCallerIsEmpty(t *testing.T) {
	fm := newTestManager(t)
	f := publish(t, fm, composeForm(t, fm, "Mine", "council", []string{"ward-1"}, 1))
	answerAll(t, fm, f.ID, "alice")

	rows, raw := answeredAs(t, mountAsCaller(t, fm), "", "")
	if len(rows) != 0 || strings.TrimSpace(raw) != "[]" {
		t.Fatalf("no caller answered %s", raw)
	}
}

func TestAnsweredKeepsTheFirstAnswerTimeAndShowsTheLastEdit(t *testing.T) {
	fm := newTestManager(t)
	f := publish(t, fm, composeForm(t, fm, "Twice", "council", []string{"ward-1"}, 1))
	answerAll(t, fm, f.ID, "alice")
	first, _ := answeredAs(t, mountAsCaller(t, fm), "alice", "")
	answerAll(t, fm, f.ID, "alice")

	rows, _ := answeredAs(t, mountAsCaller(t, fm), "alice", "")
	if len(rows) != 1 || rows[0].AnsweredAt != first[0].AnsweredAt || rows[0].EditedAt == "" {
		t.Fatalf("after a resubmission: %+v (first %+v)", rows, first)
	}
}

func TestAnsweredStillListsAClosedFormForAResidentMount(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}}))
	f := publish(t, fm, composeForm(t, fm, "Closed", "council", []string{"ward-1"}, 1))
	answerAll(t, fm, f.ID, "alice")
	if _, err := fm.Close(context.Background(), f.ID, "sender-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(fm, routes.Mount{Caller: waitingCallerFrom, ReachedOnly: true}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), waitingCallerKey{}, r.Header.Get("X-Test-Caller"))))
	})
	rows, _ := answeredAs(t, h, "alice", "")
	if titlesOf(rows) != "Closed" || rows[0].Open {
		t.Fatalf("resident sees %+v", rows)
	}
}

func TestAnsweredDoesNotShadowReadingAFormByID(t *testing.T) {
	fm := newTestManager(t)
	f := seedForm(t, fm)
	r := httptest.NewRequest(http.MethodGet, "/forms/"+f.ID, nil)
	rec := httptest.NewRecorder()
	mountAsCaller(t, fm).ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
