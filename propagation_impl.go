// propagation_impl.go — network rollup/pick-up tracking. Ported from
// mwanachama-backend-api-gateway's
// internal/store/memory/survey_store_public.go's propagation half.
package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/gormstore"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// Rollup summarizes a form's propagation. Stalled is exactly "targeted but
// PickedUpAt is still unset" — no time-based staleness threshold.
func (m *formManager) Rollup(ctx context.Context, formID string) (models.Rollup, error) {
	var rows []gormstore.PropagationRow
	if err := m.db.WithContext(ctx).Table(m.tables.Propagation).Where("form_id = ?", formID).Find(&rows).Error; err != nil {
		return models.Rollup{}, fmt.Errorf("Rollup: %w", err)
	}
	var out models.Rollup
	for _, r := range rows {
		out.Targeted++
		if r.PickedUpAt != "" {
			out.PickedUp++
		} else {
			out.Stalled++
			out.StalledChapterIDs = append(out.StalledChapterIDs, r.ChapterID)
		}
	}
	sort.Strings(out.StalledChapterIDs)
	return out, nil
}

// PickUp records a chapter picking up a form it was targeted with. Does not
// create a Propagation row — one must already exist from Publish. Idempotent
// on an already-picked-up row.
func (m *formManager) PickUp(ctx context.Context, formID, chapterID string) (models.Propagation, error) {
	var row gormstore.PropagationRow
	err := m.db.WithContext(ctx).Table(m.tables.Propagation).Where("form_id = ? AND chapter_id = ?", formID, chapterID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Propagation{}, ErrTargetNotFound
		}
		return models.Propagation{}, fmt.Errorf("PickUp: %w", err)
	}
	if row.PickedUpAt != "" {
		return gormstore.PropagationFromRow(row), nil
	}
	now := models.NowRFC3339()
	err = m.db.WithContext(ctx).Table(m.tables.Propagation).Where("id = ?", row.ID).
		Updates(map[string]any{"state": string(models.PropagationPickedUp), "picked_up_at": now}).Error
	if err != nil {
		return models.Propagation{}, fmt.Errorf("PickUp: %w", err)
	}
	row.State, row.PickedUpAt = string(models.PropagationPickedUp), now
	return gormstore.PropagationFromRow(row), nil
}

// ListPropagation returns a form's propagation rows, TargetedAt-then-id order.
func (m *formManager) ListPropagation(ctx context.Context, formID string) ([]models.Propagation, error) {
	var rows []gormstore.PropagationRow
	if err := m.db.WithContext(ctx).Table(m.tables.Propagation).Where("form_id = ?", formID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListPropagation: %w", err)
	}
	out := make([]models.Propagation, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.PropagationFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TargetedAt != out[j].TargetedAt {
			return out[i].TargetedAt < out[j].TargetedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
