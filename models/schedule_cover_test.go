package models

import (
	"errors"
	"testing"
	"time"
)

func weeklyOnMondays() FormSchedule {
	return FormSchedule{Start: "2026-01-05T09:00:00Z", Zone: "UTC", Rule: "FREQ=WEEKLY;BYDAY=MO", OpenDays: 3}
}

func at(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAScheduleRepeatsOnlyWhenItSaysSo(t *testing.T) {
	if (FormSchedule{}).Repeats() {
		t.Error("an empty schedule repeats")
	}
	for name, s := range map[string]FormSchedule{
		"start":    {Start: "2026-01-05T09:00:00Z"},
		"rule":     {Rule: "FREQ=DAILY"},
		"dates":    {Dates: []string{"2026-01-05"}},
		"zone":     {Zone: "UTC"},
		"end":      {End: "2026-02-01T00:00:00Z"},
		"openDays": {OpenDays: 2},
	} {
		if !s.Repeats() {
			t.Errorf("%s alone should make it repeat", name)
		}
	}
}

func TestAScheduleIsRefusedWhenItCannotBeRead(t *testing.T) {
	for name, s := range map[string]FormSchedule{
		"no zone":      {Start: "2026-01-05T09:00:00Z"},
		"unknown zone": {Start: "2026-01-05T09:00:00Z", Zone: "Mars/Olympus"},
		"bad start":    {Start: "monday", Zone: "UTC"},
		"bad rule":     {Start: "2026-01-05T09:00:00Z", Zone: "UTC", Rule: "FREQ=SOMETIMES"},
		"dates no end": {Start: "2026-01-05T09:00:00Z", Zone: "UTC", Dates: []string{"2026-01-12"}},
	} {
		if err := s.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := weeklyOnMondays().Validate(); err != nil {
		t.Errorf("a weekly schedule: %v", err)
	}
}

func TestThePeriodAtATimeIsTheOneThatOpenedLastAndHasNotClosed(t *testing.T) {
	s := weeklyOnMondays()
	p, err := s.PeriodAt(at("2026-01-14T10:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Key != "2026-01-12" || !p.OpensAt.Equal(at("2026-01-12T09:00:00Z")) || !p.ClosesAt.Equal(at("2026-01-15T09:00:00Z")) {
		t.Errorf("period: %+v", p)
	}
	if _, err := s.PeriodAt(at("2026-01-16T10:00:00Z")); !errors.Is(err, ErrNoPeriod) {
		t.Errorf("after the open days, between periods: %v", err)
	}
	if _, err := s.PeriodAt(at("2026-01-15T09:00:00Z")); !errors.Is(err, ErrNoPeriod) {
		t.Errorf("at the closing instant the period is closed: %v", err)
	}
	if _, err := s.PeriodAt(at("2026-01-01T00:00:00Z")); !errors.Is(err, ErrNoPeriod) {
		t.Errorf("before the first period: %v", err)
	}
}

func TestThePeriodKeyIsTheOpeningDayInTheSchedulesOwnZone(t *testing.T) {
	s := FormSchedule{Start: "2026-01-05T23:30:00Z", Zone: "Africa/Nairobi", Rule: "FREQ=WEEKLY;BYDAY=TU", OpenDays: 2}
	p, err := s.PeriodAt(at("2026-01-06T22:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Key != "2026-01-06" {
		t.Errorf("key: %q", p.Key)
	}
}

func TestThePreviousPeriodIsTheOneBeforeTheCurrentOne(t *testing.T) {
	s := weeklyOnMondays()
	prev, err := s.PreviousPeriod(at("2026-01-14T10:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if prev.Key != "2026-01-05" {
		t.Errorf("previous: %+v", prev)
	}
	if _, err := s.PreviousPeriod(at("2026-01-06T10:00:00Z")); !errors.Is(err, ErrNoPeriod) {
		t.Errorf("the first period has no earlier one: %v", err)
	}
	if _, err := s.PreviousPeriod(at("2026-01-16T10:00:00Z")); !errors.Is(err, ErrNoPeriod) {
		t.Errorf("between periods there is no current one: %v", err)
	}
	if _, err := (FormSchedule{}).PreviousPeriod(at("2026-01-14T10:00:00Z")); err == nil {
		t.Error("an unreadable schedule has no previous period")
	}
}

func TestAnUnreadableScheduleHasNoPeriodAndNoRecurrence(t *testing.T) {
	bad := FormSchedule{Start: "nope", Zone: "UTC"}
	if _, err := bad.PeriodAt(time.Now()); err == nil {
		t.Error("period of a bad schedule")
	}
	if _, err := bad.Recur(); err == nil {
		t.Error("recurrence of a bad schedule")
	}
	if _, err := weeklyOnMondays().Recur(); err != nil {
		t.Errorf("recurrence of a weekly schedule: %v", err)
	}
}
