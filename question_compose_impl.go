package mwanachamaforms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) draftQuestion(ctx context.Context, formID, questionID string) (models.Question, error) {
	f, err := m.Get(ctx, formID)
	if err != nil {
		return models.Question{}, err
	}
	if f.Status != models.StatusDraft {
		return models.Question{}, ErrNotDraft
	}
	q, err := m.findQuestion(ctx, questionID)
	if err != nil {
		return models.Question{}, err
	}
	if q.FormID != formID {
		return models.Question{}, ErrQuestionNotFound
	}
	return q, nil
}

func (m *formManager) RemoveQuestion(ctx context.Context, formID, questionID string) error {
	if _, err := m.draftQuestion(ctx, formID, questionID); err != nil {
		return err
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.deleteWhere(tx, roleQuestionOption, "question_id = ?", questionID); err != nil {
			return err
		}
		if err := m.deleteWhere(tx, roleQuestion, "id = ?", questionID); err != nil {
			return err
		}
		remaining, err := m.orderedQuestionIDs(tx, formID)
		if err != nil {
			return err
		}
		return m.renumber(tx, remaining)
	})
	if err != nil {
		return fmt.Errorf("RemoveQuestion: %w", err)
	}
	return nil
}

func (m *formManager) MoveQuestion(ctx context.Context, formID, questionID string, position int) ([]models.Question, error) {
	if _, err := m.draftQuestion(ctx, formID, questionID); err != nil {
		return nil, err
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ids, err := m.orderedQuestionIDs(tx, formID)
		if err != nil {
			return err
		}
		if position < 1 || position > len(ids) {
			return fmt.Errorf("%w: position must be between 1 and %d", ErrInvalidQuestion, len(ids))
		}
		rest := make([]string, 0, len(ids))
		for _, id := range ids {
			if id != questionID {
				rest = append(rest, id)
			}
		}
		moved := append(append(append([]string{}, rest[:position-1]...), questionID), rest[position-1:]...)
		return m.renumber(tx, moved)
	})
	if err != nil {
		return nil, fmt.Errorf("MoveQuestion: %w", err)
	}
	return m.ListQuestions(ctx, formID)
}

func (m *formManager) orderedQuestionIDs(tx *gorm.DB, formID string) ([]string, error) {
	var ids []string
	err := tx.Table(m.table(roleQuestion)).Where("form_id = ?", formID).Order("ordinal").Order("id").Pluck("id", &ids).Error
	return ids, err
}

func (m *formManager) renumber(tx *gorm.DB, ids []string) error {
	for i, id := range ids {
		if err := tx.Table(m.table(roleQuestion)).Where("id = ?", id).Update("ordinal", i+1).Error; err != nil {
			return err
		}
	}
	return nil
}
