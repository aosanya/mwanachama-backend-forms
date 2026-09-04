package gormstore

import (
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PublicLinkRow is the GORM row for a [models.PublicLink]. Key is unique —
// ResolveLinkKey looks a link up by it directly.
type PublicLinkRow struct {
	ID        string `gorm:"primaryKey"`
	FormID    string `gorm:"index"`
	Key       string `gorm:"uniqueIndex"`
	Label     string
	Status    string
	CreatedAt string
	RetiredAt string
	CreatedBy string
}

func (r *PublicLinkRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RespondentRow is the GORM row for a [models.Respondent]. PublicKey is
// unique when set — UpsertRespondent looks a respondent up by it.
type RespondentRow struct {
	ID                string `gorm:"primaryKey"`
	PublicKey         string `gorm:"uniqueIndex"`
	FirstSeenAt       string
	ClaimedByMemberID string
	ClaimedAt         string
}

func (r *RespondentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// DeclarationRow is the GORM row for a [models.Declaration]. Unique on
// (respondent_id, form_id) — Declare overwrites in place on a repeat.
type DeclarationRow struct {
	ID                string `gorm:"primaryKey"`
	RespondentID      string `gorm:"uniqueIndex:idx_declaration_respondent_form"`
	FormID            string `gorm:"uniqueIndex:idx_declaration_respondent_form"`
	DeclaredChapterID string
	DeclaredText      string
	CreatedAt         string
}

func (r *DeclarationRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// PublicLinkToRow converts a domain PublicLink to its row shape.
func PublicLinkToRow(l models.PublicLink) PublicLinkRow {
	return PublicLinkRow{
		ID:        l.ID,
		FormID:    l.FormID,
		Key:       l.Key,
		Label:     l.Label,
		Status:    string(l.Status),
		CreatedAt: l.CreatedAt,
		RetiredAt: l.RetiredAt,
		CreatedBy: l.CreatedBy,
	}
}

// PublicLinkFromRow converts a row back to the domain PublicLink.
func PublicLinkFromRow(r PublicLinkRow) models.PublicLink {
	return models.PublicLink{
		ID:        r.ID,
		FormID:    r.FormID,
		Key:       r.Key,
		Label:     r.Label,
		Status:    models.LinkStatus(r.Status),
		CreatedAt: r.CreatedAt,
		RetiredAt: r.RetiredAt,
		CreatedBy: r.CreatedBy,
	}
}

// RespondentToRow converts a domain Respondent to its row shape.
func RespondentToRow(r models.Respondent) RespondentRow {
	return RespondentRow{
		ID:                r.ID,
		PublicKey:         r.PublicKey,
		FirstSeenAt:       r.FirstSeenAt,
		ClaimedByMemberID: r.ClaimedByMemberID,
		ClaimedAt:         r.ClaimedAt,
	}
}

// RespondentFromRow converts a row back to the domain Respondent.
func RespondentFromRow(r RespondentRow) models.Respondent {
	return models.Respondent{
		ID:                r.ID,
		PublicKey:         r.PublicKey,
		FirstSeenAt:       r.FirstSeenAt,
		ClaimedByMemberID: r.ClaimedByMemberID,
		ClaimedAt:         r.ClaimedAt,
	}
}

// DeclarationToRow converts a domain Declaration to its row shape.
func DeclarationToRow(d models.Declaration) DeclarationRow {
	return DeclarationRow{
		ID:                d.ID,
		RespondentID:      d.RespondentID,
		FormID:            d.FormID,
		DeclaredChapterID: d.DeclaredChapterID,
		DeclaredText:      d.DeclaredText,
		CreatedAt:         d.CreatedAt,
	}
}

// DeclarationFromRow converts a row back to the domain Declaration.
func DeclarationFromRow(r DeclarationRow) models.Declaration {
	return models.Declaration{
		ID:                r.ID,
		RespondentID:      r.RespondentID,
		FormID:            r.FormID,
		DeclaredChapterID: r.DeclaredChapterID,
		DeclaredText:      r.DeclaredText,
		CreatedAt:         r.CreatedAt,
	}
}
