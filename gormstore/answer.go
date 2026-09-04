package gormstore

import (
	"encoding/json"

	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AnswerRow is the GORM row for a [models.Answer]. Unique on (question_id,
// member_id) — mirrors the gateway's survey_answer table exactly; a resubmit
// updates the existing row rather than violating the constraint (see
// form_impl.go's SubmitAnswers doc).
type AnswerRow struct {
	ID         string `gorm:"primaryKey"`
	FormID     string `gorm:"index"`
	QuestionID string `gorm:"uniqueIndex:idx_answer_question_member"`
	MemberID   string `gorm:"uniqueIndex:idx_answer_question_member"`
	ChapterID  string
	OptionIDs  datatypes.JSON

	ValueText   string
	ValueNumber *float64
	ValueDate   string
	ValueTime   string
	ValueBool   *bool

	AnsweredAt string
	EditedAt   string
}

func (r *AnswerRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// AnswerToRow converts a domain Answer to its row shape.
func AnswerToRow(a models.Answer) AnswerRow {
	ids, _ := json.Marshal(a.OptionIDs)
	return AnswerRow{
		ID:          a.ID,
		FormID:      a.FormID,
		QuestionID:  a.QuestionID,
		MemberID:    a.MemberID,
		ChapterID:   a.ChapterID,
		OptionIDs:   datatypes.JSON(ids),
		ValueText:   a.ValueText,
		ValueNumber: a.ValueNumber,
		ValueDate:   a.ValueDate,
		ValueTime:   a.ValueTime,
		ValueBool:   a.ValueBool,
		AnsweredAt:  a.AnsweredAt,
		EditedAt:    a.EditedAt,
	}
}

// AnswerFromRow converts a row back to the domain Answer.
func AnswerFromRow(r AnswerRow) models.Answer {
	var ids []string
	if len(r.OptionIDs) > 0 {
		_ = json.Unmarshal(r.OptionIDs, &ids)
	}
	return models.Answer{
		ID:          r.ID,
		FormID:      r.FormID,
		QuestionID:  r.QuestionID,
		MemberID:    r.MemberID,
		ChapterID:   r.ChapterID,
		OptionIDs:   ids,
		ValueText:   r.ValueText,
		ValueNumber: r.ValueNumber,
		ValueDate:   r.ValueDate,
		ValueTime:   r.ValueTime,
		ValueBool:   r.ValueBool,
		AnsweredAt:  r.AnsweredAt,
		EditedAt:    r.EditedAt,
	}
}
