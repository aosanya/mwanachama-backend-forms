package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) writeSubmission(tx *gorm.DB, s sendToKeep, repeatOf, now string) (string, error) {
	var held models.Submission
	q := m.from(tx, roleSubmission).Where("form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ? AND repeat_of = ? AND withdrawn_at = ''", s.form.ID, s.memberID, s.chapterID, s.period, repeatOf)
	err := m.take(q, roleSubmission, &held, ErrSubmissionNotFound)
	switch {
	case err == nil && s.form.LockedAfterSend:
		return "", ErrLockedAfterSend
	case err == nil:
		held.EditedAt = now
		held.Collector, held.CollectedAt, held.Channel, held.Language = s.collector, now, s.channel, s.language
		return held.ID, m.replaceTx(tx, roleSubmission, held.ID, held)
	case errors.Is(err, ErrSubmissionNotFound):
		sent := models.Submission{
			ID: newID(), FormID: s.form.ID, MemberID: s.memberID, ChapterID: s.chapterID, PeriodKey: s.period, RepeatOf: repeatOf, SubmittedAt: now,
			Collector: s.collector, CollectedAt: now, Channel: s.channel, Language: s.language,
			VoiceKey: voiceKeyOf(s, repeatOf),
		}
		return sent.ID, m.insertTx(tx, roleSubmission, sent)
	default:
		return "", err
	}
}

func voiceKeyOf(s sendToKeep, repeatOf string) string {
	if s.form.OneVoice == models.OneVoiceOff || repeatOf != "" {
		return ""
	}
	return s.form.ID + "|" + s.memberID + "|" + s.period
}

func (m *formManager) GetSubmission(ctx context.Context, formID, memberID string) (models.Submission, error) {
	if memberID == "" {
		return models.Submission{}, ErrSubmissionNotFound
	}
	f, err := m.Get(ctx, formID)
	if err != nil {
		return models.Submission{}, fmt.Errorf("GetSubmission: %w", err)
	}
	period, err := currentPeriodKey(f, time.Now())
	if err != nil {
		return models.Submission{}, fmt.Errorf("GetSubmission: %w", err)
	}
	held, err := m.ListSubmissions(ctx, formID, memberID, period)
	if err != nil {
		return models.Submission{}, fmt.Errorf("GetSubmission: %w", err)
	}
	for _, s := range held {
		if s.PeriodKey == period {
			return s, nil
		}
	}
	return models.Submission{}, ErrSubmissionNotFound
}

func (m *formManager) ListSubmissions(ctx context.Context, formID, memberID, periodKey string) ([]models.Submission, error) {
	q := m.q(ctx, roleSubmission).Where("form_id = ? AND withdrawn_at = ''", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	if periodKey != "" {
		q = q.Where("period_key = ?", periodKey)
	}
	out, err := listOf[models.Submission](m, q, roleSubmission)
	if err != nil {
		return nil, fmt.Errorf("ListSubmissions: %w", err)
	}
	counts, err := m.answerCounts(ctx, formID, memberID)
	if err != nil {
		return nil, fmt.Errorf("ListSubmissions: %w", err)
	}
	for i := range out {
		out[i].Answers = counts[sentAs{out[i].MemberID, out[i].ChapterID, out[i].PeriodKey, out[i].RepeatOf}]
		out[i].Paid = out[i].PaidAt != ""
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SubmittedAt != out[j].SubmittedAt {
			return out[i].SubmittedAt < out[j].SubmittedAt
		}
		if out[i].MemberID != out[j].MemberID {
			return out[i].MemberID < out[j].MemberID
		}
		return out[i].ChapterID < out[j].ChapterID
	})
	return out, nil
}

type sentAs struct {
	MemberID  string
	ChapterID string
	PeriodKey string
	RepeatOf  string
}

func (m *formManager) answerCounts(ctx context.Context, formID, memberID string) (map[sentAs]int, error) {
	type tally struct {
		MemberID  string
		ChapterID string
		PeriodKey string
		RepeatOf  string
		Given     int
	}
	q := m.q(ctx, roleAnswer).Select("member_id, chapter_id, period_key, repeat_of, count(*) as given").Where("form_id = ? AND withdrawn_at = ''", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	var rows []tally
	if err := q.Group("member_id, chapter_id, period_key, repeat_of").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[sentAs]int, len(rows))
	for _, r := range rows {
		out[sentAs{r.MemberID, r.ChapterID, r.PeriodKey, r.RepeatOf}] = r.Given
	}
	return out, nil
}

func (m *formManager) submissionsBy(ctx context.Context, memberID string) ([]models.Submission, error) {
	return listOf[models.Submission](m, m.q(ctx, roleSubmission).Where("member_id = ? AND withdrawn_at = ''", memberID), roleSubmission)
}

func (m *formManager) sendChannel(ctx context.Context, memberID, formID string) (string, error) {
	came, err := m.cameByLink(ctx, memberID, formID)
	if err != nil || !came {
		return models.ChannelApp, err
	}
	return models.ChannelLink, nil
}

func (m *formManager) refuseResend(ctx context.Context, formID, memberID, chapterID, period string) error {
	var sent int64
	err := m.q(ctx, roleSubmission).
		Where("form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ? AND withdrawn_at = ''", formID, memberID, chapterID, period).
		Count(&sent).Error
	if err != nil {
		return fmt.Errorf("SubmitAnswers: %w", err)
	}
	if sent == 0 {
		return nil
	}
	withdrawn, err := m.sendWithdrawn(ctx, formID, memberID, chapterID, period)
	if err != nil {
		return err
	}
	if withdrawn {
		return nil
	}
	return ErrLockedAfterSend
}

func (m *formManager) sendWithdrawn(ctx context.Context, formID, memberID, chapterID, period string) (bool, error) {
	var live, hidden int64
	slot := "form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ? AND repeat_of = ''"
	if err := m.q(ctx, roleAnswer).Where(slot+" AND withdrawn_at = ''", formID, memberID, chapterID, period).Count(&live).Error; err != nil {
		return false, fmt.Errorf("SubmitAnswers: %w", err)
	}
	if err := m.q(ctx, roleAnswer).Where(slot+" AND withdrawn_at <> ''", formID, memberID, chapterID, period).Count(&hidden).Error; err != nil {
		return false, fmt.Errorf("SubmitAnswers: %w", err)
	}
	return live == 0 && hidden > 0, nil
}
