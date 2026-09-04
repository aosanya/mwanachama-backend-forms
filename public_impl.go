// public_impl.go — public link resolution, respondents, and answers.
// Ported from mwanachama-backend-api-gateway's
// internal/store/memory/survey_store_public.go and survey_store_answer.go.
package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/gormstore"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// ResolveLinkKey resolves a key to its link and form. Every failure path —
// unknown key, retired link, missing form, or a form that is not currently
// open and public — returns found=false with no error, a deliberately
// indistinguishable refusal (an allow-list of exactly the one live state,
// not a deny-list of reasons to refuse).
func (m *formManager) ResolveLinkKey(ctx context.Context, key string) (models.PublicLink, models.Form, bool, error) {
	var lrow gormstore.PublicLinkRow
	err := m.db.WithContext(ctx).Table(m.tables.PublicLinks).Where("key = ?", key).First(&lrow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PublicLink{}, models.Form{}, false, nil
		}
		return models.PublicLink{}, models.Form{}, false, fmt.Errorf("ResolveLinkKey: %w", err)
	}
	link := gormstore.PublicLinkFromRow(lrow)
	if link.Status != models.LinkActive {
		return models.PublicLink{}, models.Form{}, false, nil
	}
	f, err := m.Get(ctx, link.FormID)
	if err != nil {
		if errors.Is(err, ErrFormNotFound) {
			return models.PublicLink{}, models.Form{}, false, nil
		}
		return models.PublicLink{}, models.Form{}, false, fmt.Errorf("ResolveLinkKey: %w", err)
	}
	if f.Status != models.StatusOpen || f.Audience != models.AudiencePublic {
		return models.PublicLink{}, models.Form{}, false, nil
	}
	return link, f, true, nil
}

// UpsertRespondent looks a respondent up by PublicKey, returning the
// existing row as-is if found (a true get, not a merge) or creating one.
func (m *formManager) UpsertRespondent(ctx context.Context, in models.Respondent) (models.Respondent, error) {
	if in.PublicKey != "" {
		var row gormstore.RespondentRow
		err := m.db.WithContext(ctx).Table(m.tables.Respondents).Where("public_key = ?", in.PublicKey).First(&row).Error
		if err == nil {
			return gormstore.RespondentFromRow(row), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Respondent{}, fmt.Errorf("UpsertRespondent: %w", err)
		}
	}
	if in.FirstSeenAt == "" {
		in.FirstSeenAt = models.NowRFC3339()
	}
	row := gormstore.RespondentToRow(in)
	if err := m.db.WithContext(ctx).Table(m.tables.Respondents).Create(&row).Error; err != nil {
		return models.Respondent{}, fmt.Errorf("UpsertRespondent: %w", err)
	}
	return gormstore.RespondentFromRow(row), nil
}

// SubmitAnswers validates every answer before writing any, then upserts
// each on (QuestionID, MemberID) — a resubmit keeps the original
// AnsweredAt/ChapterID and stamps EditedAt; capture time and chapter never
// change on edit.
func (m *formManager) SubmitAnswers(ctx context.Context, formID, memberID, chapterID string, in []models.Answer) ([]models.Answer, error) {
	f, err := m.Get(ctx, formID)
	if err != nil {
		return nil, err
	}
	if f.Status != models.StatusOpen {
		return nil, ErrFormNotOpen
	}

	questions := map[string]models.Question{}
	optionsByQuestion := map[string][]models.QuestionOption{}
	for _, a := range in {
		if _, ok := questions[a.QuestionID]; ok {
			continue
		}
		var qrow gormstore.QuestionRow
		err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("id = ?", a.QuestionID).First(&qrow).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrQuestionNotOnForm
			}
			return nil, fmt.Errorf("SubmitAnswers: %w", err)
		}
		q := gormstore.QuestionFromRow(qrow)
		if q.FormID != formID {
			return nil, ErrQuestionNotOnForm
		}
		questions[a.QuestionID] = q
		if q.AnswerType.HasOptions() {
			opts, err := m.ListOptions(ctx, a.QuestionID)
			if err != nil {
				return nil, fmt.Errorf("SubmitAnswers: %w", err)
			}
			optionsByQuestion[a.QuestionID] = opts
		}
	}
	for _, a := range in {
		if err := models.ValidateAnswer(questions[a.QuestionID], optionsByQuestion[a.QuestionID], a); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAnswer, err)
		}
	}

	now := models.NowRFC3339()
	out := make([]models.Answer, 0, len(in))
	for _, a := range in {
		a.FormID = formID
		a.MemberID = memberID

		var existing gormstore.AnswerRow
		err := m.db.WithContext(ctx).Table(m.tables.Answers).Where("question_id = ? AND member_id = ?", a.QuestionID, memberID).First(&existing).Error
		switch {
		case err == nil:
			a.AnsweredAt, a.ChapterID, a.EditedAt = existing.AnsweredAt, existing.ChapterID, now
			row := gormstore.AnswerToRow(a)
			row.ID = existing.ID
			if err := m.db.WithContext(ctx).Table(m.tables.Answers).Where("id = ?", existing.ID).Save(&row).Error; err != nil {
				return nil, fmt.Errorf("SubmitAnswers: %w", err)
			}
			out = append(out, gormstore.AnswerFromRow(row))
		case errors.Is(err, gorm.ErrRecordNotFound):
			a.AnsweredAt, a.ChapterID, a.EditedAt = now, chapterID, ""
			row := gormstore.AnswerToRow(a)
			if err := m.db.WithContext(ctx).Table(m.tables.Answers).Create(&row).Error; err != nil {
				return nil, fmt.Errorf("SubmitAnswers: %w", err)
			}
			out = append(out, gormstore.AnswerFromRow(row))
		default:
			return nil, fmt.Errorf("SubmitAnswers: %w", err)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return questions[out[i].QuestionID].Ordinal < questions[out[j].QuestionID].Ordinal
	})
	return out, nil
}

// ListAnswers returns a member's answers on a form, question-Ordinal order.
func (m *formManager) ListAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error) {
	var rows []gormstore.AnswerRow
	if err := m.db.WithContext(ctx).Table(m.tables.Answers).Where("form_id = ? AND member_id = ?", formID, memberID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListAnswers: %w", err)
	}
	out := make([]models.Answer, 0, len(rows))
	ordinals := map[string]int{}
	for _, r := range rows {
		a := gormstore.AnswerFromRow(r)
		out = append(out, a)
		if _, ok := ordinals[a.QuestionID]; !ok {
			var qrow gormstore.QuestionRow
			if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("id = ?", a.QuestionID).First(&qrow).Error; err == nil {
				ordinals[a.QuestionID] = qrow.Ordinal
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return ordinals[out[i].QuestionID] < ordinals[out[j].QuestionID] })
	return out, nil
}

// Declare records a respondent's self-reported chapter for a form, unique
// per (RespondentID, FormID) — a repeat call overwrites in place.
func (m *formManager) Declare(ctx context.Context, in models.Declaration) (models.Declaration, error) {
	var existing gormstore.DeclarationRow
	err := m.db.WithContext(ctx).Table(m.tables.Declarations).Where("respondent_id = ? AND form_id = ?", in.RespondentID, in.FormID).First(&existing).Error
	switch {
	case err == nil:
		in.ID, in.CreatedAt = existing.ID, existing.CreatedAt
		row := gormstore.DeclarationToRow(in)
		if err := m.db.WithContext(ctx).Table(m.tables.Declarations).Where("id = ?", existing.ID).Save(&row).Error; err != nil {
			return models.Declaration{}, fmt.Errorf("Declare: %w", err)
		}
		return gormstore.DeclarationFromRow(row), nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		if in.CreatedAt == "" {
			in.CreatedAt = models.NowRFC3339()
		}
		row := gormstore.DeclarationToRow(in)
		if err := m.db.WithContext(ctx).Table(m.tables.Declarations).Create(&row).Error; err != nil {
			return models.Declaration{}, fmt.Errorf("Declare: %w", err)
		}
		return gormstore.DeclarationFromRow(row), nil
	default:
		return models.Declaration{}, fmt.Errorf("Declare: %w", err)
	}
}
