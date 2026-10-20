package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

var (
	errNoLink        = errors.New("mwanachamaforms: no such public link")
	errNoRespondent  = errors.New("mwanachamaforms: no such respondent")
	errNoAnswer      = errors.New("mwanachamaforms: no such answer")
	errNoDeclaration = errors.New("mwanachamaforms: no such declaration")
)

func (m *formManager) ResolveLinkKey(ctx context.Context, key string) (models.PublicLink, models.Form, bool, error) {
	var link models.PublicLink
	err := m.take(m.q(ctx, rolePublicLink).Where("key = ?", key), rolePublicLink, &link, errNoLink)
	if errors.Is(err, errNoLink) {
		return models.PublicLink{}, models.Form{}, false, nil
	}
	if err != nil {
		return models.PublicLink{}, models.Form{}, false, fmt.Errorf("ResolveLinkKey: %w", err)
	}
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

func (m *formManager) OpenPublicLink(ctx context.Context, key string) (models.ResolvedLink, error) {
	link, f, found, err := m.ResolveLinkKey(ctx, key)
	if err != nil {
		return models.ResolvedLink{}, err
	}
	if !found {
		return models.ResolvedLink{}, ErrLinkNotFound
	}
	return models.ResolvedLink{Link: link, Form: f}, nil
}

func (m *formManager) RegisterRespondent(ctx context.Context, key, publicKey string) (models.RespondentPublicView, error) {
	if _, err := m.OpenPublicLink(ctx, key); err != nil {
		return models.RespondentPublicView{}, err
	}
	r, err := m.UpsertRespondent(ctx, models.Respondent{PublicKey: publicKey})
	if err != nil {
		return models.RespondentPublicView{}, err
	}
	return r.PublicView(), nil
}

func (m *formManager) DeclareChapter(ctx context.Context, key, respondentID, declaredChapterID, declaredText string) (models.Declaration, error) {
	opened, err := m.OpenPublicLink(ctx, key)
	if err != nil {
		return models.Declaration{}, err
	}
	return m.Declare(ctx, models.Declaration{
		RespondentID:      respondentID,
		FormID:            opened.Form.ID,
		DeclaredChapterID: declaredChapterID,
		DeclaredText:      declaredText,
	})
}

func (m *formManager) UpsertRespondent(ctx context.Context, in models.Respondent) (models.Respondent, error) {
	if in.PublicKey != "" {
		var held models.Respondent
		q := m.q(ctx, roleRespondent).Where("public_key = ?", in.PublicKey)
		err := m.take(q, roleRespondent, &held, errNoRespondent)
		if err == nil {
			return held, nil
		}
		if !errors.Is(err, errNoRespondent) {
			return models.Respondent{}, fmt.Errorf("UpsertRespondent: %w", err)
		}
	}
	in.ID = newID()
	in.FirstSeenAt = models.NowRFC3339()
	in.ClaimedByMemberID, in.ClaimedAt = "", ""
	if err := m.insert(ctx, roleRespondent, in); err != nil {
		return models.Respondent{}, fmt.Errorf("UpsertRespondent: %w", err)
	}
	return in, nil
}

func (m *formManager) SubmitAnswers(ctx context.Context, formID, memberID string, in []models.Answer) ([]models.Answer, error) {
	if memberID == "" {
		return nil, fmt.Errorf("%w: nobody is answering", ErrInvalidAnswer)
	}
	f, err := m.Get(ctx, formID)
	if err != nil {
		return nil, err
	}
	if f.Status != models.StatusOpen {
		return nil, ErrFormNotOpen
	}
	if f.OutsideWindow(time.Now()) {
		return nil, ErrOutsideWindow
	}

	questions, options, err := m.questionsAnswered(ctx, formID, in)
	if err != nil {
		return nil, err
	}
	for _, a := range in {
		if err := models.ValidateAnswer(questions[a.QuestionID], options[a.QuestionID], a); err != nil {
			return nil, fmt.Errorf("%w: question %s: %v", ErrInvalidAnswer, a.QuestionID, err)
		}
	}
	if err := m.requiredAnswered(ctx, formID, in); err != nil {
		return nil, err
	}

	chapterID, err := m.aimedAt(ctx, formID)
	if err != nil {
		return nil, fmt.Errorf("SubmitAnswers: %w", err)
	}
	out := make([]models.Answer, 0, len(in))
	for _, a := range in {
		written, err := m.writeAnswer(ctx, formID, memberID, chapterID, a)
		if err != nil {
			return nil, fmt.Errorf("SubmitAnswers: %w", err)
		}
		out = append(out, written)
	}
	sort.Slice(out, func(i, j int) bool {
		return questions[out[i].QuestionID].Ordinal < questions[out[j].QuestionID].Ordinal
	})
	return out, nil
}

func (m *formManager) requiredAnswered(ctx context.Context, formID string, in []models.Answer) error {
	all, err := listOf[models.Question](m,
		m.q(ctx, roleQuestion).Where("form_id = ?", formID).Order("ordinal"), roleQuestion)
	if err != nil {
		return fmt.Errorf("SubmitAnswers: %w", err)
	}
	if missing := models.UnansweredRequired(all, in); len(missing) > 0 {
		return fmt.Errorf("%w: question %s: an answer is required", ErrInvalidAnswer, missing[0].ID)
	}
	return nil
}

func (m *formManager) questionsAnswered(ctx context.Context, formID string, in []models.Answer) (map[string]models.Question, map[string][]models.QuestionOption, error) {
	questions := map[string]models.Question{}
	options := map[string][]models.QuestionOption{}
	for _, a := range in {
		if _, held := questions[a.QuestionID]; held {
			continue
		}
		q, err := m.findQuestion(ctx, a.QuestionID)
		if err != nil {
			if errors.Is(err, ErrQuestionNotFound) {
				return nil, nil, ErrQuestionNotOnForm
			}
			return nil, nil, fmt.Errorf("SubmitAnswers: %w", err)
		}
		if q.FormID != formID {
			return nil, nil, ErrQuestionNotOnForm
		}
		questions[a.QuestionID] = q
		if !q.AnswerType.HasOptions() {
			continue
		}
		opts, err := m.ListOptions(ctx, a.QuestionID)
		if err != nil {
			return nil, nil, fmt.Errorf("SubmitAnswers: %w", err)
		}
		options[a.QuestionID] = opts
	}
	return questions, options, nil
}

func (m *formManager) writeAnswer(ctx context.Context, formID, memberID, chapterID string, a models.Answer) (models.Answer, error) {
	a.FormID, a.MemberID = formID, memberID

	var held models.Answer
	q := m.q(ctx, roleAnswer).Where("question_id = ? AND member_id = ?", a.QuestionID, memberID)
	err := m.take(q, roleAnswer, &held, errNoAnswer)
	switch {
	case err == nil:
		a.ID = held.ID
		a.AnsweredAt, a.ChapterID, a.EditedAt = held.AnsweredAt, held.ChapterID, models.NowRFC3339()
		return a, m.replace(ctx, roleAnswer, a.ID, a)
	case errors.Is(err, errNoAnswer):
		a.ID = newID()
		a.AnsweredAt, a.ChapterID, a.EditedAt = models.NowRFC3339(), chapterID, ""
		return a, m.insert(ctx, roleAnswer, a)
	default:
		return models.Answer{}, err
	}
}

func (m *formManager) ListAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error) {
	out, err := m.answersInPromptOrder(ctx,
		m.q(ctx, roleAnswer).Where("form_id = ? AND member_id = ?", formID, memberID))
	if err != nil {
		return nil, fmt.Errorf("ListAnswers: %w", err)
	}
	return out, nil
}

func (m *formManager) ListAllAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error) {
	q := m.q(ctx, roleAnswer).Where("form_id = ?", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	out, err := m.answersInPromptOrder(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("ListAllAnswers: %w", err)
	}
	return out, nil
}

func (m *formManager) answersInPromptOrder(ctx context.Context, q *gorm.DB) ([]models.Answer, error) {
	out, err := listOf[models.Answer](m, q, roleAnswer)
	if err != nil {
		return nil, err
	}

	ordinals := map[string]int{}
	for _, a := range out {
		if _, held := ordinals[a.QuestionID]; held {
			continue
		}
		if q, err := m.findQuestion(ctx, a.QuestionID); err == nil {
			ordinals[a.QuestionID] = q.Ordinal
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MemberID != out[j].MemberID {
			return out[i].MemberID < out[j].MemberID
		}
		return ordinals[out[i].QuestionID] < ordinals[out[j].QuestionID]
	})
	return out, nil
}

func (m *formManager) Declare(ctx context.Context, in models.Declaration) (models.Declaration, error) {
	var respondents int64
	if err := m.q(ctx, roleRespondent).Where("id = ?", in.RespondentID).Count(&respondents).Error; err != nil {
		return models.Declaration{}, fmt.Errorf("Declare: %w", err)
	}
	if respondents == 0 {
		return models.Declaration{}, fmt.Errorf("%w: respondent %q", ErrInvalidReference, in.RespondentID)
	}

	var held models.Declaration
	q := m.q(ctx, roleDeclaration).Where("respondent_id = ? AND form_id = ?", in.RespondentID, in.FormID)
	err := m.take(q, roleDeclaration, &held, errNoDeclaration)
	switch {
	case err == nil:
		in.ID, in.CreatedAt = held.ID, held.CreatedAt
		return in, m.replace(ctx, roleDeclaration, in.ID, in)
	case errors.Is(err, errNoDeclaration):
		in.ID = newID()
		in.CreatedAt = models.NowRFC3339()
		return in, m.insert(ctx, roleDeclaration, in)
	default:
		return models.Declaration{}, fmt.Errorf("Declare: %w", err)
	}
}
