package mwanachamaforms

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type TranslatedReader interface {
	GetIn(ctx context.Context, formID, language string) (models.Form, error)
	ListQuestionsIn(ctx context.Context, formID, language string) ([]models.Question, error)
	ListOptionsIn(ctx context.Context, questionID, language string) ([]models.QuestionOption, error)
	PublicLinkQuestionsIn(ctx context.Context, key, language string) ([]models.LinkedQuestion, error)
}

type reviewedTexts map[[2]string]string

func (t reviewedTexts) over(subjectID, field string, original *string) {
	if text, ok := t[[2]string{subjectID, field}]; ok {
		*original = text
	}
}

func (m *formManager) reviewedTextsOf(ctx context.Context, formID, language string) (reviewedTexts, error) {
	texts := reviewedTexts{}
	if language == "" {
		return texts, nil
	}
	reviewed, err := m.ReviewedTranslations(ctx, formID, language)
	if err != nil {
		return nil, err
	}
	for _, tr := range reviewed {
		texts[[2]string{tr.SubjectID, tr.Field}] = tr.Text
	}
	return texts, nil
}

func (m *formManager) GetIn(ctx context.Context, formID, language string) (models.Form, error) {
	f, err := m.Get(ctx, formID)
	if err != nil {
		return f, err
	}
	texts, err := m.reviewedTextsOf(ctx, f.ID, language)
	if err != nil {
		return models.Form{}, fmt.Errorf("GetIn: %w", err)
	}
	texts.over(f.ID, "title", &f.Title)
	return f, nil
}

func (m *formManager) ListQuestionsIn(ctx context.Context, formID, language string) ([]models.Question, error) {
	questions, err := m.ListQuestions(ctx, formID)
	if err != nil || len(questions) == 0 {
		return questions, err
	}
	texts, err := m.reviewedTextsOf(ctx, formID, language)
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsIn: %w", err)
	}
	for i := range questions {
		texts.over(questions[i].ID, "prompt", &questions[i].Prompt)
	}
	return questions, nil
}

func (m *formManager) ListOptionsIn(ctx context.Context, questionID, language string) ([]models.QuestionOption, error) {
	options, err := m.ListOptions(ctx, questionID)
	if err != nil || len(options) == 0 || language == "" {
		return options, err
	}
	q, err := m.findQuestion(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("ListOptionsIn: %w", err)
	}
	texts, err := m.reviewedTextsOf(ctx, q.FormID, language)
	if err != nil {
		return nil, fmt.Errorf("ListOptionsIn: %w", err)
	}
	overOptions(texts, options)
	return options, nil
}

func (m *formManager) PublicLinkQuestionsIn(ctx context.Context, key, language string) ([]models.LinkedQuestion, error) {
	linked, err := m.PublicLinkQuestions(ctx, key)
	if err != nil || len(linked) == 0 || language == "" {
		return linked, err
	}
	texts, err := m.reviewedTextsOf(ctx, linked[0].Question.FormID, language)
	if err != nil {
		return nil, fmt.Errorf("PublicLinkQuestionsIn: %w", err)
	}
	for i := range linked {
		texts.over(linked[i].Question.ID, "prompt", &linked[i].Question.Prompt)
		overOptions(texts, linked[i].Options)
	}
	return linked, nil
}

func overOptions(texts reviewedTexts, options []models.QuestionOption) {
	for i := range options {
		texts.over(options[i].ID, "label", &options[i].Label)
	}
}
