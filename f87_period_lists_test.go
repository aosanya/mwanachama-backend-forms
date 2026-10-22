package mwanachamaforms_test

import (
	"context"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestASendInAnEarlierPeriodLeavesTheMemberWaitingInThisOne(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	mgr.UseGroups(func(context.Context, string) ([]string, error) { return []string{"ward-1"}, nil })
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "last week"}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	prev, err := weeklySchedule().PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	moveAnswersToPeriod(t, db, prev.Key)

	waiting, err := mgr.ListWaiting(ctx, "alice")
	if err != nil || len(waiting) != 1 || waiting[0].Answered || len(waiting[0].AnsweredAs) != 0 {
		t.Fatalf("waiting = %+v, %v, want the form unanswered this period", waiting, err)
	}
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "this week"}}); err != nil {
		t.Fatalf("SubmitAnswers this period: %v", err)
	}
	waiting, err = mgr.ListWaiting(ctx, "alice")
	if err != nil || len(waiting) != 1 || !waiting[0].Answered {
		t.Fatalf("waiting = %+v, %v, want the form answered this period", waiting, err)
	}
}

func TestTheAnsweredListNamesEachSendsPeriod(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "last week"}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	prev, err := weeklySchedule().PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	moveAnswersToPeriod(t, db, prev.Key)
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "this week"}}); err != nil {
		t.Fatalf("SubmitAnswers this period: %v", err)
	}
	answered, err := mgr.ListAnswered(ctx, "alice")
	if err != nil || len(answered) != 2 {
		t.Fatalf("answered = %+v, %v, want one row per period", answered, err)
	}
	if answered[0].PeriodKey != currentWeeklyKey(t) || answered[1].PeriodKey != prev.Key {
		t.Fatalf("answered = %+v, want this period first, then %s", answered, prev.Key)
	}
}

func TestTheRegisterCountsThisPeriodsRespondents(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "last week"}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	prev, err := weeklySchedule().PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	moveAnswersToPeriod(t, db, prev.Key)
	page, err := mgr.Register(ctx, mwanachamaforms.RegisterQuery{ChapterID: "council"})
	if err != nil || len(page.Rows) != 1 || page.Rows[0].Respondents != 0 {
		t.Fatalf("register = %+v, %v, want no respondents this period", page, err)
	}
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "bob", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "this week"}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	page, err = mgr.Register(ctx, mwanachamaforms.RegisterQuery{ChapterID: "council"})
	if err != nil || len(page.Rows) != 1 || page.Rows[0].Respondents != 1 {
		t.Fatalf("register = %+v, %v, want one respondent this period", page, err)
	}
}
