package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) repeatOfFor(ctx context.Context, f models.Form, memberID, period string) (string, error) {
	if f.OneVoice == models.OneVoiceOff {
		return "", nil
	}
	var voice models.Submission
	q := m.q(ctx, roleSubmission).Where("form_id = ? AND member_id = ? AND period_key = ? AND repeat_of = '' AND withdrawn_at = ''", f.ID, memberID, period)
	err := m.take(q, roleSubmission, &voice, ErrSubmissionNotFound)
	switch {
	case errors.Is(err, ErrSubmissionNotFound):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("SubmitAnswers: %w", err)
	case f.OneVoice == models.OneVoiceRefuse:
		return "", ErrSecondVoice
	}
	return voice.ID, nil
}
