package mwanachamaforms

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ReleaseMember(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	err := m.q(ctx, roleRespondent).Where("claimed_by_member_id = ?", memberID).
		Updates(map[string]any{"claimed_by_member_id": "", "claimed_at": nil}).Error
	if err != nil {
		return fmt.Errorf("ReleaseMember: %w", err)
	}
	return nil
}

func (m *formManager) HideMemberAnswers(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	err := m.q(ctx, roleAnswer).Where("member_id = ? AND withdrawn_at = ''", memberID).
		Updates(map[string]any{"withdrawn_at": models.NowRFC3339()}).Error
	if err != nil {
		return fmt.Errorf("HideMemberAnswers: %w", err)
	}
	return nil
}

func (m *formManager) RestoreMemberAnswers(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	held, err := listOf[models.Answer](m, m.q(ctx, roleAnswer).Where("member_id = ?", memberID), roleAnswer)
	if err != nil {
		return fmt.Errorf("RestoreMemberAnswers: %w", err)
	}
	for _, id := range answersToRestore(held) {
		err := m.q(ctx, roleAnswer).Where("id = ?", id).Updates(map[string]any{"withdrawn_at": ""}).Error
		if err != nil {
			return fmt.Errorf("RestoreMemberAnswers: %w", err)
		}
	}
	return nil
}

type answerSlot struct{ questionID, chapterID, periodKey, repeatOf string }

func slotOf(a models.Answer) answerSlot {
	return answerSlot{a.QuestionID, a.ChapterID, a.PeriodKey, a.RepeatOf}
}

func answersToRestore(held []models.Answer) []string {
	live := map[answerSlot]bool{}
	latestHidden := map[answerSlot]models.Answer{}
	for _, a := range held {
		slot := slotOf(a)
		if a.WithdrawnAt == "" {
			live[slot] = true
			continue
		}
		if best, seen := latestHidden[slot]; !seen || a.WithdrawnAt > best.WithdrawnAt {
			latestHidden[slot] = a
		}
	}
	var ids []string
	for slot, a := range latestHidden {
		if !live[slot] {
			ids = append(ids, a.ID)
		}
	}
	return ids
}

func (m *formManager) ListHiddenAnswers(ctx context.Context, memberID string) ([]models.Answer, error) {
	if memberID == "" {
		return nil, fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	out, err := m.answersInPromptOrder(ctx, m.q(ctx, roleAnswer).Where("member_id = ? AND withdrawn_at <> ''", memberID))
	if err != nil {
		return nil, fmt.Errorf("ListHiddenAnswers: %w", err)
	}
	return out, nil
}
