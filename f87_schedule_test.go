package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
