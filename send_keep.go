package mwanachamaforms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/gormutil"
)

const sendAttempts = 5

type sendToKeep struct {
	form      models.Form
	memberID  string
	chapterID string
	period    string
	channel   string
	language  string
	answers   []models.Answer
}

type sendKept struct {
	answers []models.Answer
	sendID  string
}

func (m *formManager) keepSend(ctx context.Context, s sendToKeep) (sendKept, error) {
	for attempt := 0; attempt < sendAttempts; attempt++ {
		kept, err := m.keepSendOnce(ctx, s)
		if !gormutil.IsUniqueViolation(err) {
			return kept, err
		}
	}
	return sendKept{}, ErrSecondVoice
}

func (m *formManager) keepSendOnce(ctx context.Context, s sendToKeep) (sendKept, error) {
	if s.form.LockedAfterSend {
		if err := m.refuseResend(ctx, s.form.ID, s.memberID, s.chapterID, s.period); err != nil {
			return sendKept{}, err
		}
	}
	repeatOf, err := m.repeatOfFor(ctx, s.form, s.memberID, s.period)
	if err != nil {
		return sendKept{}, err
	}
	kept := sendKept{answers: make([]models.Answer, 0, len(s.answers))}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		kept.answers = kept.answers[:0]
		now := models.NowRFC3339()
		for _, a := range s.answers {
			a.PeriodKey, a.RepeatOf = s.period, repeatOf
			written, err := m.writeAnswer(tx, s.form.ID, s.memberID, s.chapterID, now, s.form.LockedAfterSend, a)
			if err != nil {
				return err
			}
			kept.answers = append(kept.answers, written)
		}
		if err := m.dropAnswersNotCarried(tx, s.form.ID, s.memberID, s.chapterID, s.period, repeatOf, s.answers); err != nil {
			return err
		}
		var err error
		kept.sendID, err = m.writeSubmission(tx, s, repeatOf, now)
		return err
	})
	if err != nil {
		return sendKept{}, fmt.Errorf("SubmitAnswers: %w", err)
	}
	return kept, nil
}
