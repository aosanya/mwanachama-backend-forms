package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

const roleTranslation = "translation"

type Translator interface {
	PutTranslation(ctx context.Context, t models.Translation) (models.Translation, error)
	ListTranslations(ctx context.Context, formID string) ([]models.Translation, error)
	ReviewedTranslations(ctx context.Context, formID, language string) ([]models.Translation, error)
	TranslationLanguages(ctx context.Context, formID string) ([]string, error)
}

var errNoTranslation = errors.New("mwanachamaforms: no translation")

func (m *formManager) PutTranslation(ctx context.Context, t models.Translation) (models.Translation, error) {
	if err := translationShape(t); err != nil {
		return models.Translation{}, err
	}
	var f models.Form
	if err := m.find(ctx, roleForm, t.FormID, &f, ErrFormNotFound); err != nil {
		return models.Translation{}, err
	}
	if err := m.subjectOfForm(ctx, t); err != nil {
		return models.Translation{}, err
	}
	t.UpdatedAt = models.NowRFC3339()
	var held models.Translation
	err := m.take(m.q(ctx, roleTranslation).
		Where("subject_id = ? AND field = ? AND language = ?", t.SubjectID, t.Field, t.Language),
		roleTranslation, &held, errNoTranslation)
	switch {
	case errors.Is(err, errNoTranslation):
		t.ID = newID()
		if err := m.insert(ctx, roleTranslation, t); err != nil {
			return models.Translation{}, fmt.Errorf("PutTranslation: %w", err)
		}
		return t, nil
	case err != nil:
		return models.Translation{}, fmt.Errorf("PutTranslation: %w", err)
	}
	if held.FormID != t.FormID || held.SubjectRole != t.SubjectRole {
		return models.Translation{}, fmt.Errorf("%w: that text belongs to another instrument or object", ErrInvalidReference)
	}
	t.ID = held.ID
	if err := m.replace(ctx, roleTranslation, held.ID, t); err != nil {
		return models.Translation{}, fmt.Errorf("PutTranslation: %w", err)
	}
	return t, nil
}

func (m *formManager) subjectOfForm(ctx context.Context, t models.Translation) error {
	var count int64
	var err error
	switch t.SubjectRole {
	case models.TranslationOfForm:
		if t.SubjectID == t.FormID {
			return nil
		}
	case models.TranslationOfQuestion:
		err = m.q(ctx, roleQuestion).Where("id = ? AND form_id = ?", t.SubjectID, t.FormID).Count(&count).Error
	case models.TranslationOfOption:
		questions := m.q(ctx, roleQuestion).Select("id").Where("form_id = ?", t.FormID)
		err = m.q(ctx, roleQuestionOption).Where("id = ? AND question_id IN (?)", t.SubjectID, questions).Count(&count).Error
	}
	if err != nil {
		return fmt.Errorf("PutTranslation: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("%w: subject_id is not a %s of that instrument", ErrInvalidReference, t.SubjectRole)
	}
	return nil
}

func translationShape(t models.Translation) error {
	switch t.SubjectRole {
	case models.TranslationOfForm, models.TranslationOfQuestion, models.TranslationOfOption:
	default:
		return fmt.Errorf("%w: subject_role must be form, question or question_option", ErrInvalidReference)
	}
	if t.FormID == "" || t.SubjectID == "" || t.Field == "" || t.Language == "" {
		return fmt.Errorf("%w: form_id, subject_id, field and language are required", ErrInvalidReference)
	}
	return nil
}

func (m *formManager) ListTranslations(ctx context.Context, formID string) ([]models.Translation, error) {
	out, err := listOf[models.Translation](m, m.q(ctx, roleTranslation).
		Where("form_id = ?", formID).Order("subject_id, field, language"), roleTranslation)
	if err != nil {
		return nil, fmt.Errorf("ListTranslations: %w", err)
	}
	return out, nil
}

func (m *formManager) ReviewedTranslations(ctx context.Context, formID, language string) ([]models.Translation, error) {
	out, err := listOf[models.Translation](m, m.q(ctx, roleTranslation).
		Where("form_id = ? AND language = ? AND reviewed = ?", formID, language, true).
		Order("subject_id, field"), roleTranslation)
	if err != nil {
		return nil, fmt.Errorf("ReviewedTranslations: %w", err)
	}
	return out, nil
}

func (m *formManager) TranslationLanguages(ctx context.Context, formID string) ([]string, error) {
	var languages []string
	err := m.q(ctx, roleTranslation).
		Where("form_id = ? AND reviewed = ?", formID, true).
		Distinct("language").Pluck("language", &languages).Error
	if err != nil {
		return nil, fmt.Errorf("TranslationLanguages: %w", err)
	}
	sort.Strings(languages)
	return languages, nil
}
