package mwanachamaforms

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"

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
	stamp := map[string]any{"withdrawn_at": models.NowRFC3339()}
	for _, role := range []string{roleAnswer, roleSubmission} {
		err := m.q(ctx, role).Where("member_id = ? AND withdrawn_at = ''", memberID).Updates(stamp).Error
		if err != nil {
			return fmt.Errorf("HideMemberAnswers: %w", err)
		}
	}
	return nil
}

func (m *formManager) RestoreMemberAnswers(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	answers, err := listOf[models.Answer](m, m.q(ctx, roleAnswer).Where("member_id = ?", memberID), roleAnswer)
	if err != nil {
		return fmt.Errorf("RestoreMemberAnswers: %w", err)
	}
	sends, err := listOf[models.Submission](m, m.q(ctx, roleSubmission).Where("member_id = ?", memberID), roleSubmission)
	if err != nil {
		return fmt.Errorf("RestoreMemberAnswers: %w", err)
	}
	sendIDs, heldBack := sendsToRestore(sends)
	answerIDs := answersToRestore(answers, heldBack)
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.unhide(tx, roleAnswer, answerIDs); err != nil {
			return err
		}
		return m.unhide(tx, roleSubmission, sendIDs)
	})
	if err != nil {
		return fmt.Errorf("RestoreMemberAnswers: %w", err)
	}
	return nil
}

func (m *formManager) unhide(tx *gorm.DB, role string, ids []string) error {
	for _, id := range ids {
		if err := m.from(tx, role).Where("id = ?", id).Updates(map[string]any{"withdrawn_at": ""}).Error; err != nil {
			return err
		}
	}
	return nil
}

type restoreSlot struct{ formID, memberID, chapterID, periodKey, repeatOf string }

func answerSlotOf(a models.Answer) restoreSlot {
	return restoreSlot{a.FormID, a.MemberID, a.ChapterID, a.PeriodKey, a.RepeatOf}
}

func sendSlotOf(s models.Submission) restoreSlot {
	return restoreSlot{s.FormID, s.MemberID, s.ChapterID, s.PeriodKey, s.RepeatOf}
}

type answerSlot struct{ questionID, chapterID, periodKey, repeatOf string }

func slotOf(a models.Answer) answerSlot {
	return answerSlot{a.QuestionID, a.ChapterID, a.PeriodKey, a.RepeatOf}
}

func answersToRestore(held []models.Answer, heldBack map[restoreSlot]bool) []string {
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
		if !live[slot] && !heldBack[answerSlotOf(a)] {
			ids = append(ids, a.ID)
		}
	}
	return ids
}

func sendsToRestore(held []models.Submission) ([]string, map[restoreSlot]bool) {
	live := map[restoreSlot]bool{}
	liveVoices := map[string]bool{}
	latestHidden := map[restoreSlot]models.Submission{}
	for _, s := range held {
		slot := sendSlotOf(s)
		if s.WithdrawnAt == "" {
			live[slot] = true
			if s.VoiceKey != "" {
				liveVoices[s.VoiceKey] = true
			}
			continue
		}
		if best, seen := latestHidden[slot]; !seen || s.WithdrawnAt > best.WithdrawnAt {
			latestHidden[slot] = s
		}
	}
	newestOfVoice := map[string]models.Submission{}
	for slot, s := range latestHidden {
		if live[slot] || s.VoiceKey == "" {
			continue
		}
		if best, seen := newestOfVoice[s.VoiceKey]; !seen || s.WithdrawnAt > best.WithdrawnAt {
			newestOfVoice[s.VoiceKey] = s
		}
	}
	var ids []string
	heldBack := map[restoreSlot]bool{}
	for slot, s := range latestHidden {
		switch {
		case live[slot]:
		case s.VoiceKey != "" && (liveVoices[s.VoiceKey] || newestOfVoice[s.VoiceKey].ID != s.ID):
			heldBack[slot] = true
		default:
			ids = append(ids, s.ID)
		}
	}
	return ids, heldBack
}

func (m *formManager) ListHiddenSends(ctx context.Context, memberID string) ([]models.Submission, error) {
	if memberID == "" {
		return nil, fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	out, err := listOf[models.Submission](m, m.q(ctx, roleSubmission).Where("member_id = ? AND withdrawn_at <> ''", memberID), roleSubmission)
	if err != nil {
		return nil, fmt.Errorf("ListHiddenSends: %w", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SubmittedAt < out[j].SubmittedAt })
	return out, nil
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
