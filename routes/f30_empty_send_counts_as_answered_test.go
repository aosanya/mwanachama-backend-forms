package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func emptySendFixture(t *testing.T) (mwanachamaforms.FormManager, mwanachamaforms.Form, mwanachamaforms.Question, http.Handler) {
	t.Helper()
	fm := newTestManager(t)
	fm.UseGroups(func(context.Context, string) ([]string, error) { return []string{"ward-1"}, nil })
	f, q := aimedOpenForm(t, fm, "ward-1")
	return fm, f, q, mountAsCaller(t, fm)
}

func sendAs(t *testing.T, h http.Handler, caller, formID, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/forms/"+formID+"/answers", strings.NewReader(body))
	r.Header.Set("X-Test-Caller", caller)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func getAs(t *testing.T, h http.Handler, caller, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.Header.Set("X-Test-Caller", caller)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func submissionsOf(t *testing.T, rec *httptest.ResponseRecorder) []mwanachamaforms.Submission {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []mwanachamaforms.Submission
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return out
}

func TestAnEmptySendCountsAsAnswered(t *testing.T) {
	_, f, _, h := emptySendFixture(t)

	if rec := getAs(t, h, "alice", "/forms/"+f.ID+"/submissions/mine"); rec.Code != http.StatusNotFound {
		t.Fatalf("before any send: %d %s, want 404", rec.Code, rec.Body.String())
	}
	if rows, _ := waitingAs(t, h, "alice", ""); len(rows) != 1 || rows[0].Answered {
		t.Fatalf("before any send, waiting = %+v", rows)
	}

	rec := sendAs(t, h, "alice", f.ID, `{"answers":[]}`)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty send: %d %s, want 200 []", rec.Code, rec.Body.String())
	}

	rec = getAs(t, h, "alice", "/forms/"+f.ID+"/submissions/mine")
	var mine mwanachamaforms.Submission
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &mine) != nil {
		t.Fatalf("after an empty send: %d %s", rec.Code, rec.Body.String())
	}
	if mine.MemberID != "alice" || mine.FormID != f.ID || mine.Answers != 0 || mine.SubmittedAt == "" ||
		mine.ChapterID != "ward-1" || mine.EditedAt != "" {
		t.Fatalf("submission = %+v", mine)
	}
	if rec := getAs(t, h, "alice", "/forms/"+f.ID+"/answers/mine"); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("answers/mine after an empty send = %s, want []", rec.Body.String())
	}
	if rows, _ := waitingAs(t, h, "alice", ""); len(rows) != 1 || !rows[0].Answered {
		t.Fatalf("after an empty send, waiting = %+v", rows)
	}
	if rows, _ := answeredAs(t, h, "alice", ""); len(rows) != 1 || rows[0].FormID != f.ID || rows[0].AnsweredAt != mine.SubmittedAt {
		t.Fatalf("after an empty send, answered = %+v", rows)
	}
}

func TestTheOperatorSeesAMemberWhoSentNothing(t *testing.T) {
	_, f, q, h := emptySendFixture(t)
	sendAs(t, h, "alice", f.ID, `{"answers":[]}`)
	sendAs(t, h, "bob", f.ID, `{"answers":[{"question_id":"`+q.ID+`","value_text":"fine"}]}`)

	all := submissionsOf(t, getAs(t, h, "operator", "/forms/"+f.ID+"/submissions"))
	if len(all) != 2 || all[0].MemberID != "alice" || all[0].Answers != 0 || all[1].MemberID != "bob" || all[1].Answers != 1 {
		t.Fatalf("submissions = %+v, want alice with none then bob with one", all)
	}
	one := submissionsOf(t, getAs(t, h, "operator", "/forms/"+f.ID+"/submissions?member_id=alice"))
	if len(one) != 1 || one[0].MemberID != "alice" {
		t.Fatalf("narrowed to alice = %+v", one)
	}
}

func TestSendingAgainNeverDoublesTheSend(t *testing.T) {
	fm, f, q, h := emptySendFixture(t)
	answer := `{"answers":[{"question_id":"` + q.ID + `","value_text":"fine"}]}`
	orders := map[string][]string{
		"empty-first":  {`{"answers":[]}`, answer},
		"answer-first": {answer, `{"answers":[]}`},
	}
	for caller, order := range orders {
		for _, body := range order {
			if rec := sendAs(t, h, caller, f.ID, body); rec.Code != http.StatusOK {
				t.Fatalf("send %s: %d %s", body, rec.Code, rec.Body.String())
			}
		}
		mine := submissionsOf(t, getAs(t, h, "operator", "/forms/"+f.ID+"/submissions?member_id="+caller))
		if len(mine) != 1 || mine[0].EditedAt == "" || mine[0].Answers != 1 {
			t.Fatalf("%v: submissions = %+v, want one edited send carrying the one answer", order, mine)
		}
	}
	page, err := fm.Register(context.Background(), mwanachamaforms.RegisterQuery{})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(page.Rows) != 1 || page.Rows[0].Respondents != 2 {
		t.Fatalf("register = %+v, want two respondents", page.Rows)
	}
}

func TestReadingMySendNeverReachesAnotherMembers(t *testing.T) {
	_, f, _, h := emptySendFixture(t)
	sendAs(t, h, "bob", f.ID, `{"answers":[]}`)
	if rec := getAs(t, h, "alice", "/forms/"+f.ID+"/submissions/mine?member_id=bob"); rec.Code != http.StatusNotFound {
		t.Fatalf("alice naming bob: %d %s, want 404", rec.Code, rec.Body.String())
	}
	if rec := getAs(t, h, "", "/forms/"+f.ID+"/submissions/mine"); rec.Code != http.StatusNotFound {
		t.Fatalf("no caller: %d %s, want 404", rec.Code, rec.Body.String())
	}
}
