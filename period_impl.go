package mwanachamaforms

import (
	"errors"
	"fmt"
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

const standingSendPrefix = "send-"

func sendPeriodKey(f Form, at time.Time) (string, error) {
	if f.Standing {
		return standingSendPrefix + newID(), nil
	}
	return currentPeriodKey(f, at)
}

func sendPeriodFor(f Form, named string, at time.Time) (string, error) {
	if f.Standing || !f.Schedule.Repeats() {
		return sendPeriodKey(f, at)
	}
	p, err := f.Schedule.PeriodAt(at)
	if errors.Is(err, models.ErrNoPeriod) {
		return "", ErrOutsideWindow
	}
	if err != nil {
		return "", fmt.Errorf("SubmitAnswers: %w", err)
	}
	if named != "" && named != p.Key {
		return "", ErrPeriodClosed
	}
	return p.Key, nil
}
