package mwanachamaforms

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"gorm.io/gorm"
	"sort"
)

func (m *formManager) CorrectSubmission(ctx context.Context, formID, memberID, correctedBy string, in []models.Answer) ([]models.Answer, error) {
	if memberID == "" || correctedBy == "" || len(in) == 0 {
		return nil, fmt.Errorf("%w: a correction names who sent, who corrects and at least one answer", ErrInvalidAnswer)
	}
	held, err := m.GetSubmission(ctx, formID, memberID)
	if err != nil {
		return nil, fmt.Errorf("CorrectSubmission: %w", err)
	}
	questions, options, err := m.questionsAnswered(ctx, formID, in)
	if err != nil {
		return nil, err
	}
	for _, a := range in {
		if err := plainTextFields(ErrInvalidAnswer, a); err != nil {
			return nil, err
		}
		if err := models.ValidateAnswerWithin(questions[a.QuestionID], options[a.QuestionID], a, m.defaultMaxFiles); err != nil {
			return nil, fmt.Errorf("%w: question %s: %v", ErrInvalidAnswer, a.QuestionID, err)
		}
	}
	if err := m.checkPlaceRefs(ctx, in); err != nil {
		return nil, err
	}
	out := make([]models.Answer, 0, len(in))
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		out = out[:0]
		now := models.NowRFC3339()
		for _, a := range in {
			a.PeriodKey, a.RepeatOf = held.PeriodKey, held.RepeatOf
			written, err := m.writeAnswer(tx, formID, memberID, held.ChapterID, now, false, a)
			if err != nil {
				return err
			}
			written.SendID = held.ID
			out = append(out, written)
		}
		held.EditedAt = now
		return m.replaceTx(tx, roleSubmission, held.ID, held)
	})
	if err != nil {
		return nil, fmt.Errorf("CorrectSubmission: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		return questions[out[i].QuestionID].Ordinal < questions[out[j].QuestionID].Ordinal
	})
	return out, nil
}
