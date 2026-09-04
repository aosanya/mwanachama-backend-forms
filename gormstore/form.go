// Package gormstore holds every GORM-specific piece of this repo: row
// structs, their conversion to/from the domain types in
// mwanachama-backend-forms/models, and table migration. Nothing outside
// this package (and the root mwanachama-backend-forms package's *_impl.go
// files, which call it) needs to know GORM exists.
package gormstore

import (
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FormRow is the GORM row for a [models.Form].
type FormRow struct {
	ID                  string `gorm:"primaryKey"`
	Title               string
	OriginatorChapterID string `gorm:"index"`
	Status              string `gorm:"index"`

	OpensAt            string
	ClosesAt           string
	PublishedAt        string
	PublishedBy        string
	ClosedAt           string
	ClosedBy           string
	SubmittedAt        string
	SubmittedBy        string
	ApprovedAt         string
	ApprovedBy         string
	ResultsPublishedAt string

	CreatedBy      string
	Audience       string
	CollectionMode string

	CreatedAt string
	UpdatedAt string
}

func (r *FormRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// FormToRow converts a domain Form to its row shape, stamping UpdatedAt.
func FormToRow(f models.Form) FormRow {
	return FormRow{
		ID:                  f.ID,
		Title:               f.Title,
		OriginatorChapterID: f.OriginatorChapterID,
		Status:              string(f.Status),
		OpensAt:             f.OpensAt,
		ClosesAt:            f.ClosesAt,
		PublishedAt:         f.PublishedAt,
		PublishedBy:         f.PublishedBy,
		ClosedAt:            f.ClosedAt,
		ClosedBy:            f.ClosedBy,
		SubmittedAt:         f.SubmittedAt,
		SubmittedBy:         f.SubmittedBy,
		ApprovedAt:          f.ApprovedAt,
		ApprovedBy:          f.ApprovedBy,
		ResultsPublishedAt:  f.ResultsPublishedAt,
		CreatedBy:           f.CreatedBy,
		Audience:            string(f.Audience),
		CollectionMode:      string(f.CollectionMode),
		CreatedAt:           f.CreatedAt,
		UpdatedAt:           models.NowRFC3339(),
	}
}

// FormFromRow converts a row back to the domain Form.
func FormFromRow(r FormRow) models.Form {
	return models.Form{
		ID:                  r.ID,
		Title:               r.Title,
		OriginatorChapterID: r.OriginatorChapterID,
		Status:              models.Status(r.Status),
		OpensAt:             r.OpensAt,
		ClosesAt:            r.ClosesAt,
		PublishedAt:         r.PublishedAt,
		PublishedBy:         r.PublishedBy,
		ClosedAt:            r.ClosedAt,
		ClosedBy:            r.ClosedBy,
		SubmittedAt:         r.SubmittedAt,
		SubmittedBy:         r.SubmittedBy,
		ApprovedAt:          r.ApprovedAt,
		ApprovedBy:          r.ApprovedBy,
		ResultsPublishedAt:  r.ResultsPublishedAt,
		CreatedBy:           r.CreatedBy,
		Audience:            models.Audience(r.Audience),
		CollectionMode:      models.CollectionMode(r.CollectionMode),
		CreatedAt:           r.CreatedAt,
		LastUpdated:         r.UpdatedAt,
	}
}
