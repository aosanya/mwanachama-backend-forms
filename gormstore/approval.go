package gormstore

import (
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalRow is the GORM row for a [models.Approval].
type ApprovalRow struct {
	ID          string `gorm:"primaryKey"`
	FormID      string `gorm:"index"`
	RequestedBy string
	RequestedAt string
	DecidedBy   string
	DecidedAt   string
	Decision    string
	Note        string
}

func (r *ApprovalRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ApprovalToRow converts a domain Approval to its row shape.
func ApprovalToRow(a models.Approval) ApprovalRow {
	return ApprovalRow{
		ID:          a.ID,
		FormID:      a.FormID,
		RequestedBy: a.RequestedBy,
		RequestedAt: a.RequestedAt,
		DecidedBy:   a.DecidedBy,
		DecidedAt:   a.DecidedAt,
		Decision:    string(a.Decision),
		Note:        a.Note,
	}
}

// ApprovalFromRow converts a row back to the domain Approval.
func ApprovalFromRow(r ApprovalRow) models.Approval {
	return models.Approval{
		ID:          r.ID,
		FormID:      r.FormID,
		RequestedBy: r.RequestedBy,
		RequestedAt: r.RequestedAt,
		DecidedBy:   r.DecidedBy,
		DecidedAt:   r.DecidedAt,
		Decision:    models.Decision(r.Decision),
		Note:        r.Note,
	}
}
