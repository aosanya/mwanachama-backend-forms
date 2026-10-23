package mwanachamaforms_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func missedMux(fm mwanachamaforms.FormManager) (*http.ServeMux, map[string]string) {
	mux := http.NewServeMux()
	actions := map[string]string{}
	for _, rt := range routes.RoutesFor(fm, routes.Mount{Caller: func(context.Context) string { return "operator-1" }}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
		actions[rt.Method+" "+rt.Path] = rt.Action
	}
	return mux, actions
}

func getMissed(t *testing.T, mux *http.ServeMux, formID, key string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/forms/"+formID+"/periods/"+key+"/missed", nil))
	return rec
}

func TestWhoMissedAClosedPeriodIsTheAudienceWithNoSendInIt(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	mgr.UseGroups(func(context.Context, string) ([]string, error) { return []string{"ward-1"}, nil })
	var asked []bool
	mgr.UseMembers(func(_ context.Context, group string, descendants bool) ([]string, error) {
		asked = append(asked, descendants)
		if group != "ward-1" {
			return nil, nil
		}
		return []string{"carol", "alice", "bob"}, nil
	})
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "last week"}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	prev, err := weeklySchedule().PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	moveAnswersToPeriod(t, db, prev.Key)
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "bob", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "this week"}}); err != nil {
		t.Fatalf("SubmitAnswers this period: %v", err)
	}

	mux, actions := missedMux(mgr)
	if got := actions["GET /forms/{form_id}/periods/{period_key}/missed"]; got != "forms.submission.list_missed" {
		t.Fatalf("who missed is gated by %q, want forms.submission.list_missed", got)
	}
	rec := getMissed(t, mux, f.ID, prev.Key)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var missed []models.MissedMember
	if err := json.Unmarshal(rec.Body.Bytes(), &missed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(missed) != 2 || missed[0].MemberID != "bob" || missed[1].MemberID != "carol" || missed[0].ChapterID != "ward-1" {
		t.Fatalf("missed = %+v, want bob and carol through ward-1", missed)
	}
	if len(asked) == 0 || asked[0] {
		t.Fatalf("members asked with descendants %v, want the target's own flag false", asked)
	}
}

func TestWhoMissedRefusesAPeriodThatHasNotClosed(t *testing.T) {
	mgr := newTestManager(t)
	mgr.UseMembers(func(context.Context, string, bool) ([]string, error) { return []string{"alice"}, nil })
	f, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	current, err := weeklySchedule().PeriodAt(time.Now())
	if err != nil {
		t.Fatalf("PeriodAt: %v", err)
	}
	mux, _ := missedMux(mgr)
	if rec := getMissed(t, mux, f.ID, current.Key); rec.Code != http.StatusConflict {
		t.Fatalf("current period status = %d, want 409", rec.Code)
	}
	if rec := getMissed(t, mux, f.ID, "2999-01-04"); rec.Code != http.StatusConflict {
		t.Fatalf("future period status = %d, want 409", rec.Code)
	}
}

func TestWhoMissedRefusesAFormThatRunsOnce(t *testing.T) {
	mgr := newTestManager(t)
	f, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Once"})
	mux, _ := missedMux(mgr)
	if rec := getMissed(t, mux, f.ID, "2026-10-05"); rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestAManagerGivenNoMembersSeamAnswersNobodyMissed(t *testing.T) {
	mgr := newTestManager(t)
	f, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	prev, err := weeklySchedule().PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	missed, err := mgr.ListMissed(context.Background(), f.ID, prev.Key)
	if err != nil || len(missed) != 0 {
		t.Fatalf("missed = %+v, %v, want none", missed, err)
	}
}
