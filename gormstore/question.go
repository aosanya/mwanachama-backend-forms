package gormstore

import (
	"encoding/json"

	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// QuestionRow is the GORM row for a [models.Question]. SupersedesQuestionID
// carries no foreign key: a version chain is walked in Go (form_impl.go),
// the same "plain indexed column, no GORM association" choice
// mwanachama-backend-actor made for Group.ParentID and for the identical
// reason — Migrate scopes AutoMigrate per mounted instance via db.Table(...),
// which an association's default table-name resolution cannot follow.
type QuestionRow struct {
	ID          string `gorm:"primaryKey"`
	FormID      string `gorm:"index"`
	Ordinal     int
	AnswerType  string
	Prompt      string
	Placeholder string
	MaxLength   *int
	UnitLabel   string
	Helper      string

	CurrencyCode string
	QuickPicks   datatypes.JSON
	YesLabel     string
	NoLabel      string

	Version              int
	SupersedesQuestionID string `gorm:"index"`
}

func (r *QuestionRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// QuestionOptionRow is the GORM row for a [models.QuestionOption].
type QuestionOptionRow struct {
	ID         string `gorm:"primaryKey"`
	QuestionID string `gorm:"index"`
	Ordinal    int
	Label      string
}

func (r *QuestionOptionRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// QuestionToRow converts a domain Question to its row shape.
func QuestionToRow(q models.Question) QuestionRow {
	picks, _ := json.Marshal(q.QuickPicks)
	return QuestionRow{
		ID:                   q.ID,
		FormID:               q.FormID,
		Ordinal:              q.Ordinal,
		AnswerType:           string(q.AnswerType),
		Prompt:               q.Prompt,
		Placeholder:          q.Placeholder,
		MaxLength:            q.MaxLength,
		UnitLabel:            q.UnitLabel,
		Helper:               q.Helper,
		CurrencyCode:         q.CurrencyCode,
		QuickPicks:           datatypes.JSON(picks),
		YesLabel:             q.YesLabel,
		NoLabel:              q.NoLabel,
		Version:              q.Version,
		SupersedesQuestionID: q.SupersedesQuestionID,
	}
}

// QuestionFromRow converts a row back to the domain Question.
func QuestionFromRow(r QuestionRow) models.Question {
	var picks []float64
	if len(r.QuickPicks) > 0 {
		_ = json.Unmarshal(r.QuickPicks, &picks)
	}
	return models.Question{
		ID:                   r.ID,
		FormID:               r.FormID,
		Ordinal:              r.Ordinal,
		AnswerType:           models.AnswerType(r.AnswerType),
		Prompt:               r.Prompt,
		Placeholder:          r.Placeholder,
		MaxLength:            r.MaxLength,
		UnitLabel:            r.UnitLabel,
		Helper:               r.Helper,
		CurrencyCode:         r.CurrencyCode,
		QuickPicks:           picks,
		YesLabel:             r.YesLabel,
		NoLabel:              r.NoLabel,
		Version:              r.Version,
		SupersedesQuestionID: r.SupersedesQuestionID,
	}
}

// QuestionOptionToRow converts a domain QuestionOption to its row shape.
func QuestionOptionToRow(o models.QuestionOption) QuestionOptionRow {
	return QuestionOptionRow{ID: o.ID, QuestionID: o.QuestionID, Ordinal: o.Ordinal, Label: o.Label}
}

// QuestionOptionFromRow converts a row back to the domain QuestionOption.
func QuestionOptionFromRow(r QuestionOptionRow) models.QuestionOption {
	return models.QuestionOption{ID: r.ID, QuestionID: r.QuestionID, Ordinal: r.Ordinal, Label: r.Label}
}
