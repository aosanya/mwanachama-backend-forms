package gormstore

import (
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PropagationRow is the GORM row for a [models.Propagation]. Unique on
// (form_id, chapter_id) — Publish creates at most one per Target, PickUp
// only ever updates an existing row (see form_impl.go's PickUp doc — it
// never creates one).
type PropagationRow struct {
	ID                string `gorm:"primaryKey"`
	FormID            string `gorm:"uniqueIndex:idx_propagation_form_chapter"`
	ChapterID         string `gorm:"uniqueIndex:idx_propagation_form_chapter"`
	State             string
	TargetedAt        string
	PickedUpAt        string
	PushedAt          string
	PushedByChapterID string
	LastReminderAt    string
}

func (r *PropagationRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// PropagationToRow converts a domain Propagation to its row shape.
func PropagationToRow(p models.Propagation) PropagationRow {
	return PropagationRow{
		ID:                p.ID,
		FormID:            p.FormID,
		ChapterID:         p.ChapterID,
		State:             string(p.State),
		TargetedAt:        p.TargetedAt,
		PickedUpAt:        p.PickedUpAt,
		PushedAt:          p.PushedAt,
		PushedByChapterID: p.PushedByChapterID,
		LastReminderAt:    p.LastReminderAt,
	}
}

// PropagationFromRow converts a row back to the domain Propagation.
func PropagationFromRow(r PropagationRow) models.Propagation {
	return models.Propagation{
		ID:                r.ID,
		FormID:            r.FormID,
		ChapterID:         r.ChapterID,
		State:             models.PropagationKind(r.State),
		TargetedAt:        r.TargetedAt,
		PickedUpAt:        r.PickedUpAt,
		PushedAt:          r.PushedAt,
		PushedByChapterID: r.PushedByChapterID,
		LastReminderAt:    r.LastReminderAt,
	}
}
