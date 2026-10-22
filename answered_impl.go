package mwanachamaforms

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ListAnswered(ctx context.Context, callerID string) ([]models.AnsweredForm, error) {
	if callerID == "" {
		return []models.AnsweredForm{}, nil
	}
	sent, err := m.submissionsBy(ctx, callerID)
	if err != nil {
		return nil, fmt.Errorf("ListAnswered: %w", err)
	}
	if len(sent) == 0 {
		return []models.AnsweredForm{}, nil
	}
	given := make(map[string][]models.Submission, len(sent))
	for _, sub := range sent {
		given[sub.FormID] = append(given[sub.FormID], sub)
	}

	ids := make([]string, 0, len(given))
	for id := range given {
		ids = append(ids, id)
	}
	forms, err := listOf[models.Form](m, m.q(ctx, roleForm).Where("id IN ?", ids), roleForm)
	if err != nil {
		return nil, fmt.Errorf("ListAnswered: %w", err)
	}

	now := time.Now()
	out := make([]models.AnsweredForm, 0, len(forms))
	for _, f := range forms {
		for _, sub := range given[f.ID] {
			out = append(out, models.AnsweredForm{
				FormID:              f.ID,
				Title:               f.Title,
				OriginatorChapterID: f.OriginatorChapterID,
				PublishedBy:         f.PublishedBy,
				PublisherName:       f.PublisherName,
				PublishingAuthority: f.PublishingAuthority,
				PublishedAt:         f.PublishedAt,
				OpensAt:             f.OpensAt,
				ClosesAt:            f.ClosesAt,
				ClosedAt:            f.ClosedAt,
				Status:              f.Status,
				Open:                f.Status == models.StatusOpen && !f.OutsideWindow(now),
				ChapterID:           sub.ChapterID,
				AnsweredAt:          sub.SubmittedAt,
				EditedAt:            sub.EditedAt,
				PeriodKey:           sub.PeriodKey,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PublishedAt != out[j].PublishedAt {
			return out[i].PublishedAt > out[j].PublishedAt
		}
		if out[i].FormID != out[j].FormID {
			return out[i].FormID < out[j].FormID
		}
		if out[i].ChapterID != out[j].ChapterID {
			return out[i].ChapterID < out[j].ChapterID
		}
		return out[i].PeriodKey > out[j].PeriodKey
	})
	return out, nil
}
