package mwanachamaforms

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) PublicLinkQuestions(ctx context.Context, key string) ([]models.LinkedQuestion, error) {
	ctx = unnarrowed(ctx)
	opened, err := m.OpenPublicLink(ctx, key)
	if err != nil {
		return nil, err
	}
	questions, err := m.currentQuestions(ctx, opened.Form.ID)
	if err != nil {
		return nil, fmt.Errorf("PublicLinkQuestions: %w", err)
	}
	out := make([]models.LinkedQuestion, 0, len(questions))
	for _, q := range questions {
		options, err := listOf[models.QuestionOption](m,
			m.q(ctx, roleQuestionOption).Where("question_id = ?", q.ID).Order("ordinal"), roleQuestionOption)
		if err != nil {
			return nil, fmt.Errorf("PublicLinkQuestions: %w", err)
		}
		out = append(out, models.LinkedQuestion{Question: q, Options: options})
	}
	return out, nil
}

func (m *formManager) currentQuestions(ctx context.Context, formID string) ([]models.Question, error) {
	superseded := m.from(m.db, roleQuestion).
		Select("supersedes_question_id").Where("supersedes_question_id <> ''")
	return listOf[models.Question](m,
		m.q(ctx, roleQuestion).Where("form_id = ? AND id NOT IN (?)", formID, superseded).Order("ordinal"),
		roleQuestion)
}
