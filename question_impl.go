package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) AddQuestion(ctx context.Context, in models.QuestionDraft) (models.Question, []models.QuestionOption, error) {
	q, options := in.Question, in.Options
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
	if err := m.q(ctx, roleQuestion).Where("form_id = ?", q.FormID).Count(&count).Error; err != nil {
		return models.Question{}, nil, fmt.Errorf("AddQuestion: %w", err)
	}
	q.ID = newID()
	q.Ordinal = int(count) + 1
	q.Version = 1
	q.SupersedesQuestionID = ""
	if err := m.checks(roleQuestion, q); err != nil {
		return models.Question{}, nil, err
	}

	var out []models.QuestionOption
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.insertTx(tx, roleQuestion, q); err != nil {
			return err
		}
		out, err = m.writeOptions(tx, q.ID, options)
		return err
	})
	if err != nil {
		return models.Question{}, nil, fmt.Errorf("AddQuestion: %w", err)
	}
	return q, out, nil
}

// writeOptions numbers a question's choices from one and writes them. The
// caller's own ordinal is never honoured — the order they arrive in is the
// order they are shown.
func (m *formManager) writeOptions(tx *gorm.DB, questionID string, options []models.QuestionOption) ([]models.QuestionOption, error) {
	out := make([]models.QuestionOption, 0, len(options))
	for i, o := range options {
		o.ID = newID()
		o.QuestionID = questionID
		o.Ordinal = i + 1
		if err := m.insertTx(tx, roleQuestionOption, o); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

// UpdateQuestion replaces a draft question's wording and options wholesale —
// old options are deleted rather than merged, so an old option id is never
// reused. Identity and versioning stay this package's: id, form_id, ordinal,
// version and supersedes_question_id are not caller-settable. A form has to
// be a draft for this to run at all, so it never reaches a published form's
// options; the published-option-lock trigger is the belt behind that.
func (m *formManager) UpdateQuestion(ctx context.Context, formID, questionID string, in models.QuestionDraft) (models.Question, []models.QuestionOption, error) {
	next, options := in.Question, in.Options
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

	cur, err := m.findQuestion(ctx, questionID)
	if err != nil {
		return models.Question{}, nil, err
	}
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
	if err := m.checks(roleQuestion, cur); err != nil {
		return models.Question{}, nil, err
	}

	var out []models.QuestionOption
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := encode(m.st.Object(roleQuestion), cur)
		if err != nil {
			return err
		}
		if err := tx.Table(m.table(roleQuestion)).Where("id = ?", questionID).Updates(row).Error; err != nil {
			return err
		}
		if err := m.deleteWhere(tx, roleQuestionOption, "question_id = ?", questionID); err != nil {
			return err
		}
		out, err = m.writeOptions(tx, questionID, options)
		return err
	})
	if err != nil {
		return models.Question{}, nil, fmt.Errorf("UpdateQuestion: %w", err)
	}
	return cur, out, nil
}

func (m *formManager) findQuestion(ctx context.Context, questionID string) (models.Question, error) {
	var q models.Question
	if err := m.find(ctx, roleQuestion, questionID, &q, ErrQuestionNotFound); err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			return models.Question{}, err
		}
		return models.Question{}, fmt.Errorf("read the question: %w", err)
	}
	return q, nil
}

// ListQuestions returns every version of every question on a form, in the
// order they are asked — superseded ones included, unlike the live count the
// register reports.
func (m *formManager) ListQuestions(ctx context.Context, formID string) ([]models.Question, error) {
	out, err := listOf[models.Question](m,
		m.q(ctx, roleQuestion).Where("form_id = ?", formID).Order("ordinal"), roleQuestion)
	if err != nil {
		return nil, fmt.Errorf("ListQuestions: %w", err)
	}
	return out, nil
}

func (m *formManager) ListOptions(ctx context.Context, questionID string) ([]models.QuestionOption, error) {
	out, err := listOf[models.QuestionOption](m,
		m.q(ctx, roleQuestionOption).Where("question_id = ?", questionID).Order("ordinal"), roleQuestionOption)
	if err != nil {
		return nil, fmt.Errorf("ListOptions: %w", err)
	}
	return out, nil
}

// AddOption appends one choice to an existing question whatever state its
// form is in — the one additive content edit a published form allows, which
// the option lock permits because it refuses only UPDATE and DELETE.
func (m *formManager) AddOption(ctx context.Context, questionID, label, actorID string) (models.AddOptionResult, error) {
	if strings.TrimSpace(label) == "" {
		return models.AddOptionResult{}, ErrMissingOptionLabel
	}
	if strings.TrimSpace(questionID) == "" {
		return models.AddOptionResult{}, ErrQuestionNotFound
	}
	if _, err := m.findQuestion(ctx, questionID); err != nil {
		return models.AddOptionResult{}, err
	}

	var before int64
	if err := m.q(ctx, roleQuestionOption).Where("question_id = ?", questionID).Count(&before).Error; err != nil {
		return models.AddOptionResult{}, fmt.Errorf("AddOption: %w", err)
	}
	option := models.QuestionOption{
		ID: newID(), QuestionID: questionID, Ordinal: int(before) + 1, Label: label,
	}
	if err := m.insert(ctx, roleQuestionOption, option); err != nil {
		return models.AddOptionResult{}, fmt.Errorf("AddOption: %w", err)
	}
	return models.AddOptionResult{
		Option:        option,
		OptionsBefore: int(before),
		OptionsAfter:  int(before) + 1,
	}, nil
}

// VersionQuestion supersedes a published question with a new row: the
// predecessor and the answers already given against it are left alone, and a
// new row carries the new wording and a fresh set of choices, keeping the
// predecessor's position and answer shape. Only the version nothing has
// superseded may be superseded again, so the history is a chain and not a
// tree.
func (m *formManager) VersionQuestion(ctx context.Context, questionID, actorID string, in models.QuestionDraft) (models.VersionQuestionResult, error) {
	next, options := in.Question, in.Options
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

	pred, err := m.findQuestion(ctx, questionID)
	if err != nil {
		return models.VersionQuestionResult{}, err
	}
	f, err := m.Get(ctx, pred.FormID)
	if err != nil {
		return models.VersionQuestionResult{}, err
	}
	if f.Status == models.StatusDraft {
		return models.VersionQuestionResult{}, ErrFormNotPublished
	}

	var superseded int64
	if err := m.q(ctx, roleQuestion).Where("supersedes_question_id = ?", questionID).Count(&superseded).Error; err != nil {
		return models.VersionQuestionResult{}, fmt.Errorf("VersionQuestion: %w", err)
	}
	if superseded > 0 {
		return models.VersionQuestionResult{}, ErrNotCurrentVersion
	}

	successor := models.Question{
		ID:                   newID(),
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
	}

	var out []models.QuestionOption
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.insertTx(tx, roleQuestion, successor); err != nil {
			return err
		}
		out, err = m.writeOptions(tx, successor.ID, options)
		return err
	})
	if err != nil {
		return models.VersionQuestionResult{}, fmt.Errorf("VersionQuestion: %w", err)
	}
	return models.VersionQuestionResult{Question: successor, Options: out}, nil
}
