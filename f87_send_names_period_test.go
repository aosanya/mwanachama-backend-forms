package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestASendNamingTheOpenPeriodLandsInIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	key := currentWeeklyKey(t)
	answers, err := mgr.SubmitAnswersFor(ctx, f.ID, "alice", "", "", key, []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine"}})
	if err != nil {
		t.Fatalf("SubmitAnswersFor: %v", err)
	}
	if len(answers) != 1 || answers[0].PeriodKey != key {
		t.Fatalf("answers = %+v, want period %s", answers, key)
	}
}

func TestASendNamingAPeriodThatHasClosedIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	p, err := weeklySchedule().PreviousPeriod(time.Now())
	if err != nil {
		t.Fatalf("PreviousPeriod: %v", err)
	}
	_, err = mgr.SubmitAnswersFor(ctx, f.ID, "alice", "", "", p.Key, []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "late"}})
	if !errors.Is(err, mwanachamaforms.ErrPeriodClosed) {
		t.Fatalf("err = %v, want ErrPeriodClosed", err)
	}
	subs, err := mgr.ListSubmissions(ctx, f.ID, "alice", "")
	if err != nil || len(subs) != 0 {
		t.Fatalf("submissions = %+v, %v, want none", subs, err)
	}
}

func TestASendWhileNoPeriodIsOpenIsRefusedAsOutsideTheWindow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	start := time.Now().Add(-48 * time.Hour).UTC()
	shut := models.FormSchedule{Start: start.Format(time.RFC3339), Zone: "UTC", Rule: "FREQ=WEEKLY;INTERVAL=1", OpenDays: 1}
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly, one day open", Schedule: shut})
	_, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "between"}})
	if !errors.Is(err, mwanachamaforms.ErrOutsideWindow) {
		t.Fatalf("err = %v, want ErrOutsideWindow", err)
	}
}

func TestAPeriodKeyOnAFormThatRunsOnceIsIgnored(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Once"})
	answers, err := mgr.SubmitAnswersFor(ctx, f.ID, "alice", "", "", "2026-01-01", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "x"}})
	if err != nil || len(answers) != 1 || answers[0].PeriodKey != "" {
		t.Fatalf("answers = %+v, %v", answers, err)
	}
}
