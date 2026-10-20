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
	answers, err := listOf[models.Answer](m, m.q(ctx, roleAnswer).Where("member_id = ?", callerID), roleAnswer)
	if err != nil {
		return nil, fmt.Errorf("ListAnswered: %w", err)
	}
	given := answeredTimesByForm(answers)
	if len(given) == 0 {
		return []models.AnsweredForm{}, nil
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
		times := given[f.ID]
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
			AnsweredAt:          times.first,
			EditedAt:            times.lastEdit,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PublishedAt != out[j].PublishedAt {
			return out[i].PublishedAt > out[j].PublishedAt
		}
		return out[i].FormID < out[j].FormID
	})
	return out, nil
}

type answeredTimes struct {
	first    string
	lastEdit string
}

func answeredTimesByForm(answers []models.Answer) map[string]answeredTimes {
	out := map[string]answeredTimes{}
	for _, a := range answers {
		t, seen := out[a.FormID]
		if !seen || (a.AnsweredAt != "" && a.AnsweredAt < t.first) {
			t.first = a.AnsweredAt
		}
		if a.EditedAt > t.lastEdit {
			t.lastEdit = a.EditedAt
		}
		out[a.FormID] = t
	}
	return out
}
