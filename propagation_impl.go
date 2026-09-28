package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

// Rollup counts how far a form has spread. Stalled is exactly "reached but
// never picked up" — there is no staleness threshold, so a group that took
// it up a second later and one that never did are the only two cases.
func (m *formManager) Rollup(ctx context.Context, formID string) (models.Rollup, error) {
	rows, err := m.propagationFor(ctx, formID)
	if err != nil {
		return models.Rollup{}, fmt.Errorf("Rollup: %w", err)
	}
	var out models.Rollup
	for _, p := range rows {
		out.Targeted++
		if p.PickedUpAt != "" {
			out.PickedUp++
			continue
		}
		out.Stalled++
		out.StalledChapterIDs = append(out.StalledChapterIDs, p.ChapterID)
	}
	sort.Strings(out.StalledChapterIDs)
	return out, nil
}

// PickUp records a group taking up a form it was reached with. It never
// creates the row — publishing does that — so a group that was never reached
// is refused rather than quietly recorded. A second call is a no-op.
func (m *formManager) PickUp(ctx context.Context, formID, chapterID string) (models.Propagation, error) {
	var p models.Propagation
	q := m.q(ctx, rolePropagation).Where("form_id = ? AND chapter_id = ?", formID, chapterID)
	if err := m.take(q, rolePropagation, &p, ErrTargetNotFound); err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			return models.Propagation{}, err
		}
		return models.Propagation{}, fmt.Errorf("PickUp: %w", err)
	}
	if p.PickedUpAt != "" {
		return p, nil
	}
	p.State, p.PickedUpAt = models.PropagationPickedUp, models.NowRFC3339()
	if err := m.replace(ctx, rolePropagation, p.ID, p); err != nil {
		return models.Propagation{}, fmt.Errorf("PickUp: %w", err)
	}
	return p, nil
}

func (m *formManager) ListPropagation(ctx context.Context, formID string) ([]models.Propagation, error) {
	out, err := m.propagationFor(ctx, formID)
	if err != nil {
		return nil, fmt.Errorf("ListPropagation: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TargetedAt != out[j].TargetedAt {
			return out[i].TargetedAt < out[j].TargetedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *formManager) propagationFor(ctx context.Context, formID string) ([]models.Propagation, error) {
	return listOf[models.Propagation](m,
		m.q(ctx, rolePropagation).Where("form_id = ?", formID), rolePropagation)
}
