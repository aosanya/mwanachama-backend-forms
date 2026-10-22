package mwanachamaforms

import (
	"errors"
	"time"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func currentPeriodKey(f Form, at time.Time) (string, error) {
	return periodKey(f, at, models.FormSchedule.PeriodAt)
}

func previousPeriodKey(f Form, at time.Time) (string, error) {
	return periodKey(f, at, models.FormSchedule.PreviousPeriod)
}

func periodKey(f Form, at time.Time, find func(models.FormSchedule, time.Time) (models.Period, error)) (string, error) {
	if !f.Schedule.Repeats() {
		return "", nil
	}
	p, err := find(f.Schedule, at)
	if errors.Is(err, models.ErrNoPeriod) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return p.Key, nil
}
