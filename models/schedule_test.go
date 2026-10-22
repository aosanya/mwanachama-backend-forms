package models

import (
	"errors"
	"testing"
	"time"
)

func weekly() FormSchedule {
	return FormSchedule{Start: "2026-10-05T09:00:00+03:00", Zone: "Africa/Nairobi", Rule: "FREQ=WEEKLY;INTERVAL=1"}
}

func TestAPeriodIsKeyedByItsOpeningDayAndClosesWhenTheNextOpens(t *testing.T) {
	at := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	p, err := weekly().PeriodAt(at)
	if err != nil {
		t.Fatal(err)
	}
	if p.Key != "2026-10-12" {
		t.Fatalf("key %q", p.Key)
	}
	if !p.ClosesAt.Equal(p.OpensAt.AddDate(0, 0, 7)) {
		t.Fatalf("closes %v opens %v", p.ClosesAt, p.OpensAt)
	}
}

func TestThePreviousPeriodIsTheOneBeforeTheCurrent(t *testing.T) {
	prev, err := weekly().PreviousPeriod(time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if prev.Key != "2026-10-05" {
		t.Fatalf("key %q", prev.Key)
	}
}

func TestThereIsNoPeriodBeforeTheStartOrAfterTheEnd(t *testing.T) {
	s := weekly()
	if _, err := s.PeriodAt(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrNoPeriod) {
		t.Fatalf("before start: %v", err)
	}
	if _, err := s.PreviousPeriod(time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrNoPeriod) {
		t.Fatalf("first period has a previous: %v", err)
	}
}

func TestAListOfDatesIsASchedule(t *testing.T) {
	s := FormSchedule{Start: "2026-01-05T08:00:00+03:00", Zone: "Africa/Nairobi", Dates: []string{"2026-01-05", "2026-05-04", "2026-09-07"}}
	p, err := s.PeriodAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if p.Key != "2026-05-04" {
		t.Fatalf("key %q", p.Key)
	}
}

func TestAScheduleWithoutAZoneOrWithBothRuleAndDatesIsRefused(t *testing.T) {
	s := weekly()
	s.Zone = ""
	if s.Validate() == nil {
		t.Fatal("no zone accepted")
	}
	s = weekly()
	s.Dates = []string{"2026-10-05"}
	if s.Validate() == nil {
		t.Fatal("rule and dates accepted")
	}
}
