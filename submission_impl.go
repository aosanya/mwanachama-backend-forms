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

func (m *formManager) writeSubmission(tx *gorm.DB, formID, memberID, chapterID, period, channel, language, now string) (string, error) {
	var held models.Submission
	q := tx.Table(m.table(roleSubmission)).Where("form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ?", formID, memberID, chapterID, period)
	err := m.take(q, roleSubmission, &held, ErrSubmissionNotFound)
	switch {
	case err == nil:
		held.EditedAt = now
		held.Collector, held.CollectedAt, held.Channel, held.Language = memberID, now, channel, language
		return held.ID, m.replaceTx(tx, roleSubmission, held.ID, held)
	case errors.Is(err, ErrSubmissionNotFound):
		sent := models.Submission{
			ID: newID(), FormID: formID, MemberID: memberID, ChapterID: chapterID, PeriodKey: period, SubmittedAt: now,
			Collector: memberID, CollectedAt: now, Channel: channel, Language: language,
		}
		return sent.ID, m.insertTx(tx, roleSubmission, sent)
	default:
		return "", err
	}
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
	q := m.q(ctx, roleSubmission).Where("form_id = ?", formID)
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
		out[i].Answers = counts[sentAs{out[i].MemberID, out[i].ChapterID, out[i].PeriodKey}]
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
}

func (m *formManager) answerCounts(ctx context.Context, formID, memberID string) (map[sentAs]int, error) {
	type tally struct {
		MemberID  string
		ChapterID string
		PeriodKey string
		Given     int
	}
	q := m.q(ctx, roleAnswer).Select("member_id, chapter_id, period_key, count(*) as given").Where("form_id = ?", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	var rows []tally
	if err := q.Group("member_id, chapter_id, period_key").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[sentAs]int, len(rows))
	for _, r := range rows {
		out[sentAs{r.MemberID, r.ChapterID, r.PeriodKey}] = r.Given
	}
	return out, nil
}

func (m *formManager) submissionsBy(ctx context.Context, memberID string) ([]models.Submission, error) {
	return listOf[models.Submission](m, m.q(ctx, roleSubmission).Where("member_id = ?", memberID), roleSubmission)
}

func (m *formManager) sendChannel(ctx context.Context, memberID, formID string) (string, error) {
	came, err := m.cameByLink(ctx, memberID, formID)
	if err != nil || !came {
		return models.ChannelApp, err
	}
	return models.ChannelLink, nil
}
