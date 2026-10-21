package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) writeSubmission(tx *gorm.DB, formID, memberID, chapterID, now string) error {
	var held models.Submission
	q := tx.Table(m.table(roleSubmission)).Where("form_id = ? AND member_id = ?", formID, memberID)
	err := m.take(q, roleSubmission, &held, ErrSubmissionNotFound)
	switch {
	case err == nil:
		held.EditedAt = now
		held.Collector, held.CollectedAt, held.Channel = memberID, now, models.ChannelApp
		return m.replaceTx(tx, roleSubmission, held.ID, held)
	case errors.Is(err, ErrSubmissionNotFound):
		return m.insertTx(tx, roleSubmission, models.Submission{
			ID: newID(), FormID: formID, MemberID: memberID, ChapterID: chapterID, SubmittedAt: now,
			Collector: memberID, CollectedAt: now, Channel: models.ChannelApp,
		})
	default:
		return err
	}
}

func (m *formManager) GetSubmission(ctx context.Context, formID, memberID string) (models.Submission, error) {
	if memberID == "" {
		return models.Submission{}, ErrSubmissionNotFound
	}
	held, err := m.ListSubmissions(ctx, formID, memberID)
	if err != nil {
		return models.Submission{}, fmt.Errorf("GetSubmission: %w", err)
	}
	if len(held) == 0 {
		return models.Submission{}, ErrSubmissionNotFound
	}
	return held[0], nil
}

func (m *formManager) ListSubmissions(ctx context.Context, formID, memberID string) ([]models.Submission, error) {
	q := m.q(ctx, roleSubmission).Where("form_id = ?", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
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
		out[i].Answers = counts[out[i].MemberID]
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SubmittedAt != out[j].SubmittedAt {
			return out[i].SubmittedAt < out[j].SubmittedAt
		}
		return out[i].MemberID < out[j].MemberID
	})
	return out, nil
}

func (m *formManager) answerCounts(ctx context.Context, formID, memberID string) (map[string]int, error) {
	type tally struct {
		MemberID string
		Given    int
	}
	q := m.q(ctx, roleAnswer).Select("member_id, count(*) as given").Where("form_id = ?", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	var rows []tally
	if err := q.Group("member_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.MemberID] = r.Given
	}
	return out, nil
}

func (m *formManager) submissionsBy(ctx context.Context, memberID string) ([]models.Submission, error) {
	return listOf[models.Submission](m, m.q(ctx, roleSubmission).Where("member_id = ?", memberID), roleSubmission)
}
