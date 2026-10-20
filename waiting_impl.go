package mwanachamaforms

import (
	"context"
	"fmt"
	"sort"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type Groups func(ctx context.Context, callerID string) ([]string, error)

func (m *formManager) UseGroups(g Groups) { m.groups = g }

func (m *formManager) ListWaiting(ctx context.Context, callerID string) ([]models.WaitingForm, error) {
	if m.groups == nil || callerID == "" {
		return []models.WaitingForm{}, nil
	}
	groups, err := m.groups(ctx, callerID)
	if err != nil {
		return nil, fmt.Errorf("ListWaiting: %w", err)
	}
	if len(groups) == 0 {
		return []models.WaitingForm{}, nil
	}

	reached, err := m.reachedThrough(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("ListWaiting: %w", err)
	}
	if len(reached) == 0 {
		return []models.WaitingForm{}, nil
	}

	ids := make([]string, 0, len(reached))
	for id := range reached {
		ids = append(ids, id)
	}
	forms, err := listOf[models.Form](m,
		m.q(ctx, roleForm).Where("id IN ? AND status = ?", ids, string(models.StatusOpen)), roleForm)
	if err != nil {
		return nil, fmt.Errorf("ListWaiting: %w", err)
	}

	out := make([]models.WaitingForm, 0, len(forms))
	for _, f := range forms {
		row, err := m.waitingRow(ctx, f, callerID, reached[f.ID])
		if err != nil {
			return nil, fmt.Errorf("ListWaiting: %w", err)
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ClosesAt != out[j].ClosesAt {
			return out[i].ClosesAt < out[j].ClosesAt
		}
		return out[i].FormID < out[j].FormID
	})
	return out, nil
}

func (m *formManager) reachedThrough(ctx context.Context, groups []string) (map[string][]string, error) {
	type reach struct {
		FormID    string
		ChapterID string
	}
	var rows []reach
	for _, role := range []string{rolePropagation, roleTarget} {
		var found []reach
		if err := m.q(ctx, role).Select("form_id, chapter_id").
			Where("chapter_id IN ?", groups).Find(&found).Error; err != nil {
			return nil, err
		}
		rows = append(rows, found...)
	}

	via := map[string]map[string]bool{}
	for _, r := range rows {
		if via[r.FormID] == nil {
			via[r.FormID] = map[string]bool{}
		}
		via[r.FormID][r.ChapterID] = true
	}
	out := make(map[string][]string, len(via))
	for formID, chapters := range via {
		list := make([]string, 0, len(chapters))
		for c := range chapters {
			list = append(list, c)
		}
		sort.Strings(list)
		out[formID] = list
	}
	return out, nil
}

func (m *formManager) waitingRow(ctx context.Context, f models.Form, callerID string, via []string) (models.WaitingForm, error) {
	questions, err := m.currentQuestionCount(ctx, f.ID)
	if err != nil {
		return models.WaitingForm{}, err
	}
	var mine int64
	if err := m.q(ctx, roleAnswer).Where("form_id = ? AND member_id = ?", f.ID, callerID).
		Count(&mine).Error; err != nil {
		return models.WaitingForm{}, err
	}
	return models.WaitingForm{
		FormID:              f.ID,
		Title:               f.Title,
		OriginatorChapterID: f.OriginatorChapterID,
		PublishedBy:         f.PublishedBy,
		PublisherName:       f.PublisherName,
		PublishingAuthority: f.PublishingAuthority,
		PublishedAt:         f.PublishedAt,
		OpensAt:             f.OpensAt,
		ClosesAt:            f.ClosesAt,
		Questions:           questions,
		ReachedVia:          via,
		Answered:            mine > 0,
	}, nil
}
