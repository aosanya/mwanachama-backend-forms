// target_impl.go — Target CRUD. Ported from
// mwanachama-backend-api-gateway's
// internal/store/memory/survey_store_content.go's target half.
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

// AddTarget aims a draft, member-audience Form at a chapter. Descendant-of-
// originator validation of ChapterID is deliberately not this package's
// job — it does not depend on a chapter domain to check against (mirrors
// the gateway's own survey.Repository.AddTarget, which leaves that to its
// HTTP layer).
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

	var count int64
	if err := m.db.WithContext(ctx).Table(m.tables.Targets).Where("form_id = ? AND chapter_id = ?", t.FormID, t.ChapterID).Count(&count).Error; err != nil {
		return models.Target{}, fmt.Errorf("AddTarget: %w", err)
	}
	if count > 0 {
		return models.Target{}, ErrDuplicateTarget
	}

	if t.CreatedAt == "" {
		t.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.TargetToRow(t)
	if err := m.db.WithContext(ctx).Table(m.tables.Targets).Create(&row).Error; err != nil {
		return models.Target{}, fmt.Errorf("AddTarget: %w", err)
	}
	return gormstore.TargetFromRow(row), nil
}

// RemoveTarget removes a draft form's target.
func (m *formManager) RemoveTarget(ctx context.Context, formID, targetID string) error {
	f, err := m.Get(ctx, formID)
	if err != nil {
		return err
	}
	if f.Status != models.StatusDraft {
		return ErrNotDraft
	}

	var row gormstore.TargetRow
	err = m.db.WithContext(ctx).Table(m.tables.Targets).Where("id = ?", targetID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTargetNotFound
		}
		return fmt.Errorf("RemoveTarget: %w", err)
	}
	if row.FormID != formID {
		return ErrTargetNotFound
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Targets).Where("id = ?", targetID).Delete(&gormstore.TargetRow{}).Error; err != nil {
		return fmt.Errorf("RemoveTarget: %w", err)
	}
	return nil
}

// ListTargets returns a form's targets, created_at-then-id order.
func (m *formManager) ListTargets(ctx context.Context, formID string) ([]models.Target, error) {
	var rows []gormstore.TargetRow
	if err := m.db.WithContext(ctx).Table(m.tables.Targets).Where("form_id = ?", formID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListTargets: %w", err)
	}
	out := make([]models.Target, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.TargetFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
