package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) AddTarget(ctx context.Context, t models.Target) (models.Target, error) {
	f, err := m.Get(ctx, t.FormID)
	if err != nil {
		if errors.Is(err, ErrFormNotFound) {
			return models.Target{}, fmt.Errorf("%w: form_id", ErrInvalidReference)
		}
		return models.Target{}, err
	}
	if f.Status != models.StatusDraft {
		return models.Target{}, ErrNotDraft
	}
	if f.Audience == models.AudiencePublic {
		return models.Target{}, ErrPublicFormNoTargets
	}

	held, err := m.ListTargets(ctx, t.FormID)
	if err != nil {
		return models.Target{}, fmt.Errorf("AddTarget: %w", err)
	}
	for _, h := range held {
		if h.ChapterID == t.ChapterID {
			return models.Target{}, ErrDuplicateTarget
		}
	}

	t.ID = newID()
	t.CreatedAt = models.NowRFC3339()
	if err := m.insert(ctx, roleTarget, t); err != nil {
		return models.Target{}, fmt.Errorf("AddTarget: %w", err)
	}
	return t, nil
}

func (m *formManager) RemoveTarget(ctx context.Context, formID, targetID string) error {
	f, err := m.Get(ctx, formID)
	if err != nil {
		return err
	}
	if f.Status != models.StatusDraft {
		return ErrNotDraft
	}

	var t models.Target
	if err := m.find(ctx, roleTarget, targetID, &t, ErrTargetNotFound); err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			return err
		}
		return fmt.Errorf("RemoveTarget: %w", err)
	}
	if t.FormID != formID {
		return ErrTargetNotFound
	}
	if err := m.deleteWhere(m.db.WithContext(ctx), roleTarget, "id = ?", targetID); err != nil {
		return fmt.Errorf("RemoveTarget: %w", err)
	}
	return nil
}

func (m *formManager) ListTargets(ctx context.Context, formID string) ([]models.Target, error) {
	out, err := listOf[models.Target](m, m.q(ctx, roleTarget).Where("form_id = ?", formID), roleTarget)
	if err != nil {
		return nil, fmt.Errorf("ListTargets: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *formManager) aimedAt(ctx context.Context, formID, memberID string) (string, error) {
	held, err := m.ListTargets(ctx, formID)
	if err != nil || len(held) == 0 {
		return "", err
	}
	if len(held) == 1 || m.groups == nil {
		return held[0].ChapterID, nil
	}
	groups, err := m.groups(ctx, memberID)
	if err != nil {
		return "", err
	}
	for _, t := range held {
		if slices.Contains(groups, t.ChapterID) {
			return t.ChapterID, nil
		}
	}
	return held[0].ChapterID, nil
}
