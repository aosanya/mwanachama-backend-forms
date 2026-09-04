package gormstore

import (
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TargetRow is the GORM row for a [models.Target]. Unique on (form_id,
// chapter_id) via a composite index — form_impl.go's AddTarget pre-checks
// the same pair and returns [ErrDuplicateTarget] before this index would
// ever be hit, mirroring mwanachama-backend-actor's
// checkUniqueAttributes-then-partial-index belt-and-braces pattern.
type TargetRow struct {
	ID                  string `gorm:"primaryKey"`
	FormID              string `gorm:"uniqueIndex:idx_target_form_chapter"`
	ChapterID           string `gorm:"uniqueIndex:idx_target_form_chapter"`
	IncludesDescendants bool
	CreatedAt           string
}

func (r *TargetRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// TargetToRow converts a domain Target to its row shape.
func TargetToRow(t models.Target) TargetRow {
	return TargetRow{
		ID:                  t.ID,
		FormID:              t.FormID,
		ChapterID:           t.ChapterID,
		IncludesDescendants: t.IncludesDescendants,
		CreatedAt:           t.CreatedAt,
	}
}

// TargetFromRow converts a row back to the domain Target.
func TargetFromRow(r TargetRow) models.Target {
	return models.Target{
		ID:                  r.ID,
		FormID:              r.FormID,
		ChapterID:           r.ChapterID,
		IncludesDescendants: r.IncludesDescendants,
		CreatedAt:           r.CreatedAt,
	}
}
