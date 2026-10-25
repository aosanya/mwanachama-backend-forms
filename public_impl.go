package mwanachamaforms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	if f.Status != models.StatusOpen || f.Audience != models.AudiencePublic || f.PastClosing(time.Now()) {
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
	if err := m.checkDeclarable(ctx, opened.Form.ID, declaredChapterID); err != nil {
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
	if in.PublicKey == "" {
		key, err := mintRespondentKey()
		if err != nil {
			return models.Respondent{}, fmt.Errorf("UpsertRespondent: %w", err)
		}
		in.PublicKey = key
	}
	in.ID = newID()
	in.FirstSeenAt = models.NowRFC3339()
	in.ClaimedByMemberID, in.ClaimedAt = "", ""
	if err := m.insert(ctx, roleRespondent, in); err != nil {
		return models.Respondent{}, fmt.Errorf("UpsertRespondent: %w", err)
	}
	return in, nil
}

func mintRespondentKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func (m *formManager) SubmitAnswers(ctx context.Context, formID, memberID string, in []models.Answer) ([]models.Answer, error) {
	return m.SubmitAnswersAs(ctx, formID, memberID, "", in)
}

func (m *formManager) SubmitAnswersAs(ctx context.Context, formID, memberID, answeringAs string, in []models.Answer) ([]models.Answer, error) {
	return m.SubmitAnswersIn(ctx, formID, memberID, answeringAs, "", in)
}

func (m *formManager) SubmitAnswersIn(ctx context.Context, formID, memberID, answeringAs, language string, in []models.Answer) ([]models.Answer, error) {
	return m.SubmitAnswersFor(ctx, formID, memberID, answeringAs, language, "", in)
}

func (m *formManager) SubmitAnswersFor(ctx context.Context, formID, memberID, answeringAs, language, periodKey string, in []models.Answer) ([]models.Answer, error) {
	if memberID == "" {
		return nil, fmt.Errorf("%w: nobody is answering", ErrInvalidAnswer)
	}
	f, err := m.Get(ctx, formID)
	if err != nil {
		return nil, err
	}
	if unreached, err := m.unreachedByCaller(ctx, f); err != nil || unreached {
		if err != nil {
			return nil, err
		}
		return nil, ErrFormNotFound
	}
	if f.Status != models.StatusOpen {
		return nil, ErrFormNotOpen
	}
	if f.OutsideWindow(time.Now()) {
		return nil, ErrOutsideWindow
	}
	period, err := sendPeriodFor(f, periodKey, time.Now())
	if err != nil {
		return nil, err
	}

	questions, options, err := m.questionsAnswered(ctx, formID, in)
	if err != nil {
		return nil, err
	}
	for _, a := range in {
		if err := models.ValidateAnswerWithin(questions[a.QuestionID], options[a.QuestionID], a, m.defaultMaxFiles); err != nil {
			return nil, fmt.Errorf("%w: question %s: %v", ErrInvalidAnswer, a.QuestionID, err)
		}
	}
	if err := m.requiredAnswered(ctx, formID, in); err != nil {
		return nil, err
	}

	chapterID, err := m.sendingAs(ctx, f, memberID, answeringAs)
	if err != nil {
		return nil, err
	}
	channel, err := m.sendChannel(ctx, memberID, formID)
	if err != nil {
		return nil, fmt.Errorf("SubmitAnswers: %w", err)
	}
	out := make([]models.Answer, 0, len(in))
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := models.NowRFC3339()
		for _, a := range in {
			a.PeriodKey = period
			written, err := m.writeAnswer(tx, formID, memberID, chapterID, now, a)
			if err != nil {
				return err
			}
			out = append(out, written)
		}
		if err := m.dropAnswersNotCarried(tx, formID, memberID, chapterID, period, in); err != nil {
			return err
		}
		return m.writeSubmission(tx, formID, memberID, chapterID, period, channel, language, now)
	})
	if err != nil {
		return nil, fmt.Errorf("SubmitAnswers: %w", err)
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

func (m *formManager) writeAnswer(tx *gorm.DB, formID, memberID, chapterID, now string, a models.Answer) (models.Answer, error) {
	a.FormID, a.MemberID = formID, memberID

	var held models.Answer
	q := tx.Table(m.table(roleAnswer)).Where("question_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ?", a.QuestionID, memberID, chapterID, a.PeriodKey)
	err := m.take(q, roleAnswer, &held, errNoAnswer)
	switch {
	case err == nil:
		a.ID = held.ID
		a.AnsweredAt, a.ChapterID, a.EditedAt = held.AnsweredAt, held.ChapterID, now
		return a, m.replaceTx(tx, roleAnswer, a.ID, a)
	case errors.Is(err, errNoAnswer):
		a.ID = newID()
		a.AnsweredAt, a.ChapterID, a.EditedAt = now, chapterID, ""
		return a, m.insertTx(tx, roleAnswer, a)
	default:
		return models.Answer{}, err
	}
}

func (m *formManager) dropAnswersNotCarried(tx *gorm.DB, formID, memberID, chapterID, period string, carried []models.Answer) error {
	if len(carried) == 0 {
		return m.deleteWhere(tx, roleAnswer, "form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ?", formID, memberID, chapterID, period)
	}
	kept := make([]string, 0, len(carried))
	for _, a := range carried {
		kept = append(kept, a.QuestionID)
	}
	return m.deleteWhere(tx, roleAnswer, "form_id = ? AND member_id = ? AND chapter_id = ? AND period_key = ? AND question_id NOT IN ?", formID, memberID, chapterID, period, kept)
}

func (m *formManager) ListAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error) {
	f, err := m.Get(unnarrowed(ctx), formID)
	if err != nil {
		return nil, err
	}
	period, err := currentPeriodKey(f, time.Now())
	if err != nil {
		return nil, err
	}
	return m.answersInPeriod(ctx, formID, memberID, period)
}

func (m *formManager) ListPreviousAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error) {
	f, err := m.Get(unnarrowed(ctx), formID)
	if err != nil {
		return nil, err
	}
	period, err := previousPeriodKey(f, time.Now())
	if err != nil {
		return nil, err
	}
	if period == "" {
		return []models.Answer{}, nil
	}
	return m.answersInPeriod(ctx, formID, memberID, period)
}

func (m *formManager) answersInPeriod(ctx context.Context, formID, memberID, period string) ([]models.Answer, error) {
	return m.answersInPromptOrder(ctx,
		m.q(ctx, roleAnswer).Where("form_id = ? AND member_id = ? AND period_key = ?", formID, memberID, period))
}

func (m *formManager) ListAllAnswers(ctx context.Context, formID, memberID, periodKey string) ([]models.Answer, error) {
	q := m.q(ctx, roleAnswer).Where("form_id = ?", formID)
	if memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	if periodKey != "" {
		q = q.Where("period_key = ?", periodKey)
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
