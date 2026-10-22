package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/aosanya/mwanachama-backend-shared/recur"
)

var ErrNoPeriod = errors.New("forms: no period at that time")

const PeriodKeyLayout = "2006-01-02"

type FormSchedule struct {
	Start string   `json:"start"`
	End   string   `json:"end,omitempty"`
	Zone  string   `json:"zone"`
	Rule  string   `json:"rule,omitempty"`
	Dates []string `json:"dates,omitempty"`
}

type Period struct {
	Key      string    `json:"key"`
	OpensAt  time.Time `json:"opens_at"`
	ClosesAt time.Time `json:"closes_at,omitempty"`
}

func (f FormSchedule) Recur() (recur.Schedule, error) {
	zone, err := time.LoadLocation(f.Zone)
	if err != nil || f.Zone == "" {
		return recur.Schedule{}, fmt.Errorf("%w: unknown time zone %q", recur.ErrInvalidSchedule, f.Zone)
	}
	start, err := time.Parse(time.RFC3339, f.Start)
	if err != nil {
		return recur.Schedule{}, fmt.Errorf("%w: start %q is not RFC 3339", recur.ErrInvalidSchedule, f.Start)
	}
	s := recur.Schedule{Start: start, Zone: zone}
	if f.End != "" {
		if s.End, err = time.Parse(time.RFC3339, f.End); err != nil {
			return recur.Schedule{}, fmt.Errorf("%w: end %q is not RFC 3339", recur.ErrInvalidSchedule, f.End)
		}
	}
	if f.Rule != "" {
		rule, err := recur.ParseRule(f.Rule)
		if err != nil {
			return recur.Schedule{}, err
		}
		s.Rule = &rule
	}
	for _, raw := range f.Dates {
		d, err := recur.ParseDate(raw)
		if err != nil {
			return recur.Schedule{}, err
		}
		s.Dates = append(s.Dates, d)
	}
	return s, s.Validate()
}

func (f FormSchedule) Repeats() bool {
	return f.Start != "" || f.Rule != "" || len(f.Dates) > 0 || f.Zone != "" || f.End != ""
}

func (f FormSchedule) Validate() error {
	_, err := f.Recur()
	return err
}

func (f FormSchedule) PeriodAt(at time.Time) (Period, error) {
	s, err := f.Recur()
	if err != nil {
		return Period{}, err
	}
	opens, closes, ok := s.Period(at)
	if !ok || (!closes.IsZero() && !at.Before(closes)) {
		return Period{}, ErrNoPeriod
	}
	return Period{Key: opens.In(s.Zone).Format(PeriodKeyLayout), OpensAt: opens, ClosesAt: closes}, nil
}

func (f FormSchedule) PreviousPeriod(at time.Time) (Period, error) {
	current, err := f.PeriodAt(at)
	if err != nil {
		return Period{}, err
	}
	return f.PeriodAt(current.OpensAt.Add(-time.Nanosecond))
}
