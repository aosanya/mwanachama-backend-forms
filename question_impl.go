// question_impl.go — Question/QuestionOption CRUD and versioning. Ported
// from mwanachama-backend-api-gateway's
// internal/store/memory/survey_store_content.go, survey_option_add.go, and
// survey_question_version.go — without the custody/act-log writing those
// last two carry upstream: this repo does not depend on the gateway's
// custody domain, the same considered exclusion
// mwanachama-backend-actor already made for Deregister (see this repo's
// CLAUDE.md).
package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/gormstore"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// AddQuestion appends a new Question and its options to a draft Form.
// Ordinal, Version and SupersedesQuestionID are always server-assigned.
func (m *formManager) AddQuestion(ctx context.Context, q models.Question, options []models.QuestionOption) (models.Question, []models.QuestionOption, error) {
	f, err := m.Get(ctx, q.FormID)
	if err != nil {
		if errors.Is(err, ErrFormNotFound) {
			return models.Question{}, nil, fmt.Errorf("%w: form_id", ErrInvalidReference)
		}
		return models.Question{}, nil, err
	}
	if f.Status != models.StatusDraft {
		return models.Question{}, nil, ErrNotDraft
	}

	var count int64
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("form_id = ?", q.FormID).Count(&count).Error; err != nil {
		return models.Question{}, nil, fmt.Errorf("AddQuestion: %w", err)
	}
	q.Ordinal = int(count) + 1
	q.Version = 1
	q.SupersedesQuestionID = ""

	row := gormstore.QuestionToRow(q)
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Create(&row).Error; err != nil {
		return models.Question{}, nil, fmt.Errorf("AddQuestion: %w", err)
	}
	newQ := gormstore.QuestionFromRow(row)

	outOpts := make([]models.QuestionOption, 0, len(options))
	for i, o := range options {
		o.QuestionID = newQ.ID
		o.Ordinal = i + 1
		orow := gormstore.QuestionOptionToRow(o)
		if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Create(&orow).Error; err != nil {
			return models.Question{}, nil, fmt.Errorf("AddQuestion: %w", err)
		}
		outOpts = append(outOpts, gormstore.QuestionOptionFromRow(orow))
	}
	return newQ, outOpts, nil
}

// UpdateQuestion replaces a draft question's wording and options wholesale
// — old options are deleted, not merged, so an old option id is never
// reused. Identity/versioning fields (id, form_id, ordinal, version,
// supersedes_question_id) are never caller-settable. Because a Form must be
// StatusDraft for this to run at all, it never touches a published form's
// options — the published-option-lock trigger (gormstore's syncConstraints)
// is the belt behind this same freeze.
func (m *formManager) UpdateQuestion(ctx context.Context, formID, questionID string, next models.Question, options []models.QuestionOption) (models.Question, []models.QuestionOption, error) {
	if strings.TrimSpace(next.Prompt) == "" {
		return models.Question{}, nil, ErrMissingPrompt
	}
	for _, o := range options {
		if strings.TrimSpace(o.Label) == "" {
			return models.Question{}, nil, ErrMissingOptionLabel
		}
	}

	f, err := m.Get(ctx, formID)
	if err != nil {
		return models.Question{}, nil, err
	}
	if f.Status != models.StatusDraft {
		return models.Question{}, nil, ErrNotDraft
	}

	var row gormstore.QuestionRow
	err = m.db.WithContext(ctx).Table(m.tables.Questions).Where("id = ?", questionID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Question{}, nil, ErrQuestionNotFound
		}
		return models.Question{}, nil, fmt.Errorf("UpdateQuestion: %w", err)
	}
	cur := gormstore.QuestionFromRow(row)
	if cur.FormID != formID {
		return models.Question{}, nil, ErrQuestionNotFound
	}

	cur.Prompt = next.Prompt
	cur.AnswerType = next.AnswerType
	cur.Placeholder = next.Placeholder
	cur.MaxLength = next.MaxLength
	cur.UnitLabel = next.UnitLabel
	cur.Helper = next.Helper
	cur.CurrencyCode = next.CurrencyCode
	cur.QuickPicks = next.QuickPicks
	cur.YesLabel = next.YesLabel
	cur.NoLabel = next.NoLabel

	updRow := gormstore.QuestionToRow(cur)
	updates := map[string]any{
		"prompt":        updRow.Prompt,
		"answer_type":   updRow.AnswerType,
		"placeholder":   updRow.Placeholder,
		"max_length":    updRow.MaxLength,
		"unit_label":    updRow.UnitLabel,
		"helper":        updRow.Helper,
		"currency_code": updRow.CurrencyCode,
		"quick_picks":   updRow.QuickPicks,
		"yes_label":     updRow.YesLabel,
		"no_label":      updRow.NoLabel,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("id = ?", questionID).Updates(updates).Error; err != nil {
		return models.Question{}, nil, fmt.Errorf("UpdateQuestion: %w", err)
	}

	if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Where("question_id = ?", questionID).Delete(&gormstore.QuestionOptionRow{}).Error; err != nil {
		return models.Question{}, nil, fmt.Errorf("UpdateQuestion: %w", err)
	}
	outOpts := make([]models.QuestionOption, 0, len(options))
	for i, o := range options {
		o.QuestionID = questionID
		o.Ordinal = i + 1
		orow := gormstore.QuestionOptionToRow(o)
		if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Create(&orow).Error; err != nil {
			return models.Question{}, nil, fmt.Errorf("UpdateQuestion: %w", err)
		}
		outOpts = append(outOpts, gormstore.QuestionOptionFromRow(orow))
	}
	return cur, outOpts, nil
}

// ListQuestions returns every version of every question on a form, Ordinal
// order (includes superseded versions — unlike Register's live-question
// count).
func (m *formManager) ListQuestions(ctx context.Context, formID string) ([]models.Question, error) {
	var rows []gormstore.QuestionRow
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("form_id = ?", formID).Order("ordinal").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListQuestions: %w", err)
	}
	out := make([]models.Question, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.QuestionFromRow(r))
	}
	return out, nil
}

// ListOptions returns a question's options, Ordinal order.
func (m *formManager) ListOptions(ctx context.Context, questionID string) ([]models.QuestionOption, error) {
	var rows []gormstore.QuestionOptionRow
	if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Where("question_id = ?", questionID).Order("ordinal").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListOptions: %w", err)
	}
	out := make([]models.QuestionOption, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.QuestionOptionFromRow(r))
	}
	return out, nil
}

// AddOption appends one option to an existing question, regardless of the
// form's status — the one additive content edit this package allows on a
// published form (the published-option-lock trigger only blocks UPDATE/
// DELETE, never INSERT).
func (m *formManager) AddOption(ctx context.Context, questionID, label, actorID string) (models.AddOptionResult, error) {
	if strings.TrimSpace(label) == "" {
		return models.AddOptionResult{}, ErrMissingOptionLabel
	}
	if strings.TrimSpace(questionID) == "" {
		return models.AddOptionResult{}, ErrQuestionNotFound
	}
	var qrow gormstore.QuestionRow
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("id = ?", questionID).First(&qrow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.AddOptionResult{}, ErrQuestionNotFound
		}
		return models.AddOptionResult{}, fmt.Errorf("AddOption: %w", err)
	}

	var before int64
	if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Where("question_id = ?", questionID).Count(&before).Error; err != nil {
		return models.AddOptionResult{}, fmt.Errorf("AddOption: %w", err)
	}
	row := gormstore.QuestionOptionToRow(models.QuestionOption{QuestionID: questionID, Ordinal: int(before) + 1, Label: label})
	if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Create(&row).Error; err != nil {
		return models.AddOptionResult{}, fmt.Errorf("AddOption: %w", err)
	}
	return models.AddOptionResult{
		Option:        gormstore.QuestionOptionFromRow(row),
		OptionsBefore: int(before),
		OptionsAfter:  int(before) + 1,
	}, nil
}

// VersionQuestion supersedes a published question with a new row: the
// predecessor and its existing options/answers are left untouched, a new
// Question row carries the new wording plus a fresh option set, keeping the
// predecessor's Ordinal and AnswerType. Only the current (not yet
// superseded) version of a question may be versioned again — a chain, not a
// tree.
func (m *formManager) VersionQuestion(ctx context.Context, questionID string, next models.Question, options []models.QuestionOption, actorID string) (models.VersionQuestionResult, error) {
	if strings.TrimSpace(questionID) == "" {
		return models.VersionQuestionResult{}, ErrQuestionNotFound
	}
	if strings.TrimSpace(next.Prompt) == "" {
		return models.VersionQuestionResult{}, ErrMissingPrompt
	}
	for _, o := range options {
		if strings.TrimSpace(o.Label) == "" {
			return models.VersionQuestionResult{}, ErrMissingOptionLabel
		}
	}

	var predRow gormstore.QuestionRow
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("id = ?", questionID).First(&predRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.VersionQuestionResult{}, ErrQuestionNotFound
		}
		return models.VersionQuestionResult{}, fmt.Errorf("VersionQuestion: %w", err)
	}
	pred := gormstore.QuestionFromRow(predRow)

	f, err := m.Get(ctx, pred.FormID)
	if err != nil {
		return models.VersionQuestionResult{}, err
	}
	if f.Status == models.StatusDraft {
		return models.VersionQuestionResult{}, ErrFormNotPublished
	}

	var supersededCount int64
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("supersedes_question_id = ?", questionID).Count(&supersededCount).Error; err != nil {
		return models.VersionQuestionResult{}, fmt.Errorf("VersionQuestion: %w", err)
	}
	if supersededCount > 0 {
		return models.VersionQuestionResult{}, ErrNotCurrentVersion
	}

	row := gormstore.QuestionToRow(models.Question{
		FormID:               pred.FormID,
		Ordinal:              pred.Ordinal,
		AnswerType:           pred.AnswerType,
		Prompt:               next.Prompt,
		Placeholder:          next.Placeholder,
		MaxLength:            next.MaxLength,
		UnitLabel:            next.UnitLabel,
		Helper:               next.Helper,
		CurrencyCode:         next.CurrencyCode,
		QuickPicks:           next.QuickPicks,
		YesLabel:             next.YesLabel,
		NoLabel:              next.NoLabel,
		Version:              pred.Version + 1,
		SupersedesQuestionID: questionID,
	})
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Create(&row).Error; err != nil {
		return models.VersionQuestionResult{}, fmt.Errorf("VersionQuestion: %w", err)
	}
	newQ := gormstore.QuestionFromRow(row)

	outOpts := make([]models.QuestionOption, 0, len(options))
	for i, o := range options {
		o.QuestionID = newQ.ID
		o.Ordinal = i + 1
		orow := gormstore.QuestionOptionToRow(o)
		if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Create(&orow).Error; err != nil {
			return models.VersionQuestionResult{}, fmt.Errorf("VersionQuestion: %w", err)
		}
		outOpts = append(outOpts, gormstore.QuestionOptionFromRow(orow))
	}
	return models.VersionQuestionResult{Question: newQ, Options: outOpts}, nil
}
