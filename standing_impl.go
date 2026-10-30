package mwanachamaforms

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type StandingAudience func(ctx context.Context, callerID, groupID string) (bool, error)

func (m *formManager) UseStandingAudience(a StandingAudience) { m.standingAudience = a }

func (m *formManager) standingGroup(ctx context.Context, callerID, formID, as string) (string, error) {
	if m.standingAudience == nil || callerID == "" {
		return "", nil
	}
	held, err := m.ListTargets(ctx, formID)
	if err != nil {
		return "", err
	}
	for _, t := range held {
		if as != "" && t.ChapterID != as {
			continue
		}
		in, err := m.standingAudience(ctx, callerID, t.ChapterID)
		if err != nil {
			return "", err
		}
		if in {
			return t.ChapterID, nil
		}
	}
	return "", nil
}

func (m *formManager) sentToNobody(ctx context.Context, formID string) (bool, error) {
	var held int64
	err := m.q(ctx, roleTarget).Where("form_id = ?", formID).Count(&held).Error
	return held == 0, err
}

func (m *formManager) sendingAs(ctx context.Context, f models.Form, memberID, as string) (string, error) {
	if !f.Standing || m.standingAudience == nil {
		return m.answeringAs(ctx, f.ID, memberID, as)
	}
	nobody, err := m.sentToNobody(ctx, f.ID)
	if err != nil {
		return "", fmt.Errorf("SubmitAnswers: %w", err)
	}
	if nobody {
		return m.answeringAs(ctx, f.ID, memberID, as)
	}
	group, err := m.standingGroup(ctx, memberID, f.ID, as)
	if err != nil {
		return "", fmt.Errorf("SubmitAnswers: %w", err)
	}
	if group == "" && as != "" {
		return "", fmt.Errorf("%w: the caller is not in the audience of group %q", ErrInvalidAnswer, as)
	}
	if group == "" {
		return "", fmt.Errorf("%w: the caller is not in the audience of any group the form was sent to", ErrInvalidAnswer)
	}
	return group, nil
}

func (m *formManager) GetSend(ctx context.Context, formID, sendID, memberID string) (models.Send, error) {
	if memberID == "" || sendID == "" {
		return models.Send{}, ErrSubmissionNotFound
	}
	var held models.Submission
	q := m.q(ctx, roleSubmission).Where("id = ? AND form_id = ? AND member_id = ?", sendID, formID, memberID)
	if err := m.take(q, roleSubmission, &held, ErrSubmissionNotFound); err != nil {
		return models.Send{}, err
	}
	answers, err := m.answersInPromptOrder(ctx, m.q(ctx, roleAnswer).Where(
		"form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ?",
		formID, memberID, held.ChapterID, held.PeriodKey))
	if err != nil {
		return models.Send{}, fmt.Errorf("GetSend: %w", err)
	}
	held.Answers = len(answers)
	return models.Send{Submission: held, Answers: answers}, nil
}
