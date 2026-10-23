package models

import (
	"errors"
	"time"

	"github.com/aosanya/mwanachama-backend-shared/recur"
)

var ErrNoPeriod = errors.New("forms: no period at that time")

const PeriodKeyLayout = "2006-01-02"

type FormSchedule recur.Wire

type Period struct {
	Key      string    `json:"key"`
	OpensAt  time.Time `json:"opens_at"`
	ClosesAt time.Time `json:"closes_at,omitempty"`
}

func (f FormSchedule) Recur() (recur.Schedule, error) {
	return recur.Wire(f).Schedule()
}

func (f FormSchedule) Repeats() bool {
	return f.Start != "" || f.Rule != "" || len(f.Dates) > 0 || f.Zone != "" || f.End != "" || f.OpenDays != 0
}

func (f FormSchedule) Validate() error {
	_, err := f.Recur()
	return err
}

func (f FormSchedule) PeriodAt(at time.Time) (Period, error) {
	p, err := f.periodOpenedBy(at)
	if err != nil {
		return Period{}, err
	}
	if !p.ClosesAt.IsZero() && !at.Before(p.ClosesAt) {
		return Period{}, ErrNoPeriod
	}
	return p, nil
}

func (f FormSchedule) PreviousPeriod(at time.Time) (Period, error) {
	current, err := f.PeriodAt(at)
	if err != nil {
		return Period{}, err
	}
	return f.periodOpenedBy(current.OpensAt.Add(-time.Nanosecond))
}

func (f FormSchedule) periodOpenedBy(at time.Time) (Period, error) {
	s, err := f.Recur()
	if err != nil {
		return Period{}, err
	}
	opens, closes, ok := s.Period(at)
	if !ok {
		return Period{}, ErrNoPeriod
	}
	return Period{Key: opens.In(s.Zone).Format(PeriodKeyLayout), OpensAt: opens, ClosesAt: closes}, nil
}
