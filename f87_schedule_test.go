package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func weeklySchedule() models.FormSchedule {
	return models.FormSchedule{Start: "2026-10-05T09:00:00+03:00", Zone: "Africa/Nairobi", Rule: "FREQ=WEEKLY;INTERVAL=1"}
}

func TestAFormKeepsTheScheduleItWasComposedWith(t *testing.T) {
	um := newTestManager(t)
	ctx := context.Background()
	f, err := um.Create(ctx, mwanachamaforms.Form{Title: "Weekly check", OriginatorChapterID: "chapter-1", ClosesAt: futureRFC3339(24 * time.Hour), Schedule: weeklySchedule()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := um.Get(ctx, f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Schedule.Rule != "FREQ=WEEKLY;INTERVAL=1" || got.Schedule.Zone != "Africa/Nairobi" || got.Schedule.Start != weeklySchedule().Start {
		t.Fatalf("schedule = %+v", got.Schedule)
	}
	p, err := got.Schedule.PeriodAt(time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC))
	if err != nil || p.Key != "2026-10-12" {
		t.Fatalf("period = %+v, %v", p, err)
	}
}

func TestAFormThatRunsOnceCarriesNoSchedule(t *testing.T) {
	um := newTestManager(t)
	ctx := context.Background()
	f := newDraftForm(t, um)
	got, err := um.Get(ctx, f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Schedule.Repeats() {
		t.Fatalf("schedule = %+v", got.Schedule)
	}
}

func TestAnInvalidScheduleIsRefusedOnCreateAndUpdate(t *testing.T) {
	um := newTestManager(t)
	ctx := context.Background()
	bad := weeklySchedule()
	bad.Zone = "Nowhere/Atlantis"
	_, err := um.Create(ctx, mwanachamaforms.Form{Title: "x", OriginatorChapterID: "chapter-1", ClosesAt: futureRFC3339(time.Hour), Schedule: bad})
	if !errors.Is(err, mwanachamaforms.ErrScheduleInvalid) {
		t.Fatalf("create err = %v", err)
	}
	f := newDraftForm(t, um)
	f.Schedule = models.FormSchedule{Start: "2026-10-05T09:00:00+03:00", Zone: "Africa/Nairobi"}
	if _, err := um.Update(ctx, f); !errors.Is(err, mwanachamaforms.ErrScheduleInvalid) {
		t.Fatalf("update err = %v", err)
	}
}

func currentWeeklyKey(t *testing.T) string {
	t.Helper()
	p, err := weeklySchedule().PeriodAt(time.Now())
	if err != nil {
		t.Fatalf("PeriodAt: %v", err)
	}
	return p.Key
}

func TestASendOnARepeatingFormIsStampedWithTheCurrentPeriod(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Weekly", Schedule: weeklySchedule()})
	answers, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine", PeriodKey: "1999-01-01"}})
	if err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	want := currentWeeklyKey(t)
	if len(answers) != 1 || answers[0].PeriodKey != want {
		t.Fatalf("answers = %+v, want period %s", answers, want)
	}
	subs, err := mgr.ListSubmissions(ctx, f.ID, "alice")
	if err != nil || len(subs) != 1 || subs[0].PeriodKey != want {
		t.Fatalf("submissions = %+v, %v, want period %s", subs, err, want)
	}
}

func TestASendOnAFormThatRunsOnceCarriesNoPeriod(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Once"})
	answers, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "fine", PeriodKey: "1999-01-01"}})
	if err != nil || len(answers) != 1 || answers[0].PeriodKey != "" {
		t.Fatalf("answers = %+v, %v, want no period", answers, err)
	}
}

func moveAnswersToPeriod(t *testing.T, db *gorm.DB, key string) {
	t.Helper()
	s := testSpec(t, "test")
	for _, role := range []string{"answer", "submission"} {
		o, _ := s.ByRole(role)
		if err := db.Exec("update "+s.TableFor(o)+" set period_key = ?", key).Error; err != nil {
			t.Fatalf("move %s: %v", role, err)
		}
	}
}

func TestAMembersPreviousPeriodAnswersAreReadApartFromThisPeriods(t *testing.T) {
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

	if mine, err := mgr.ListAnswers(ctx, f.ID, "alice"); err != nil || len(mine) != 0 {
		t.Fatalf("this period's answers = %+v, %v, want none", mine, err)
	}
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "this week"}}); err != nil {
		t.Fatalf("SubmitAnswers this period: %v", err)
	}
	previous, err := mgr.ListPreviousAnswers(ctx, f.ID, "alice")
	if err != nil || len(previous) != 1 || previous[0].ValueText != "last week" || previous[0].PeriodKey != prev.Key {
		t.Fatalf("previous = %+v, %v", previous, err)
	}
	mine, err := mgr.ListAnswers(ctx, f.ID, "alice")
	if err != nil || len(mine) != 1 || mine[0].ValueText != "this week" {
		t.Fatalf("this period's answers = %+v, %v", mine, err)
	}
	if others, _ := mgr.ListPreviousAnswers(ctx, f.ID, "bob"); len(others) != 0 {
		t.Fatalf("bob's previous = %+v", others)
	}
}

func TestAFormThatRunsOnceHasNoPreviousAnswers(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Once"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	got, err := mgr.ListPreviousAnswers(ctx, f.ID, "alice")
	if err != nil || len(got) != 0 {
		t.Fatalf("previous = %+v, %v", got, err)
	}
}

func TestMySubmissionIsThisPeriodsAndCountsOnlyItsOwnAnswers(t *testing.T) {
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

	if got, err := mgr.GetSubmission(ctx, f.ID, "alice"); !errors.Is(err, mwanachamaforms.ErrSubmissionNotFound) {
		t.Fatalf("this period's submission = %+v, %v, want not found", got, err)
	}
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "this week"}}); err != nil {
		t.Fatalf("SubmitAnswers this period: %v", err)
	}
	mine, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || mine.PeriodKey == prev.Key || mine.Answers != 1 {
		t.Fatalf("this period's submission = %+v, %v", mine, err)
	}
	all, err := mgr.ListSubmissions(ctx, f.ID, "alice")
	if err != nil || len(all) != 2 {
		t.Fatalf("submissions = %+v, %v", all, err)
	}
	for _, s := range all {
		if s.Answers != 1 {
			t.Fatalf("submission %+v counts answers from another period", s)
		}
	}
}

func TestEveryonesAnswersNarrowToOnePeriod(t *testing.T) {
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

	if all, err := mgr.ListAllAnswers(ctx, f.ID, "", ""); err != nil || len(all) != 2 {
		t.Fatalf("every period = %+v, %v", all, err)
	}
	last, err := mgr.ListAllAnswers(ctx, f.ID, "", prev.Key)
	if err != nil || len(last) != 1 || last[0].ValueText != "last week" {
		t.Fatalf("last period = %+v, %v", last, err)
	}
}
