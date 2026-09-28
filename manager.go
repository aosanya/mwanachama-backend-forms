package mwanachamaforms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type (
	Form                  = models.Form
	Status                = models.Status
	Audience              = models.Audience
	CollectionMode        = models.CollectionMode
	Question              = models.Question
	QuestionOption        = models.QuestionOption
	AddOptionResult       = models.AddOptionResult
	VersionQuestionResult = models.VersionQuestionResult
	Target                = models.Target
	Rollup                = models.Rollup
	Propagation           = models.Propagation
	PropagationKind       = models.PropagationKind
	RegisterQuery         = models.RegisterQuery
	RegisterRow           = models.RegisterRow
	RegisterPage          = models.RegisterPage
	Approval              = models.Approval
	Decision              = models.Decision
	PublicLink            = models.PublicLink
	LinkStatus            = models.LinkStatus
	Respondent            = models.Respondent
	Declaration           = models.Declaration
	Answer                = models.Answer
	QuestionDraft         = models.QuestionDraft
	ResolvedLink          = models.ResolvedLink
	RespondentPublicView  = models.RespondentPublicView
	AnswerType            = models.AnswerType
)

const (
	StatusDraft     = models.StatusDraft
	StatusSubmitted = models.StatusSubmitted
	StatusApproved  = models.StatusApproved
	StatusOpen      = models.StatusOpen
	StatusClosed    = models.StatusClosed
)

const (
	AudienceMember = models.AudienceMember
	AudiencePublic = models.AudiencePublic
)

const (
	CollectionSelf        = models.CollectionSelf
	CollectionInterviewer = models.CollectionInterviewer
)

const (
	DecisionApproved = models.DecisionApproved
	DecisionRefused  = models.DecisionRefused
)

const (
	AnswerYesNo        = models.AnswerYesNo
	AnswerSingleChoice = models.AnswerSingleChoice
	AnswerFreeText     = models.AnswerFreeText
)

const (
	LinkActive = models.LinkActive
)

const (
	PropagationTargeted = models.PropagationTargeted
	PropagationPickedUp = models.PropagationPickedUp
)

const TimeLayout = models.TimeLayout

func NowRFC3339() string { return models.NowRFC3339() }

func ValidateAnswer(q Question, options []QuestionOption, a Answer) error {
	return models.ValidateAnswer(q, options, a)
}

func ValidateWindow(opensAt, closesAt string) error {
	return models.ValidateWindow(opensAt, closesAt)
}

func ValidateAudienceCollection(a Audience, m CollectionMode) error {
	return models.ValidateAudienceCollection(a, m)
}

func VisibleToMembers(s Status) bool { return models.VisibleToMembers(s) }

func Deletable(s Status) bool { return models.Deletable(s) }

type FormManager interface {
	Create(ctx context.Context, f models.Form) (models.Form, error)

	Get(ctx context.Context, id string) (models.Form, error)

	Update(ctx context.Context, f models.Form) (models.Form, error)

	ListForChapter(ctx context.Context, chapterID string) ([]models.Form, error)

	ListReachable(ctx context.Context, ancestry []string) ([]models.Form, error)

	Register(ctx context.Context, q models.RegisterQuery) (models.RegisterPage, error)

	Submit(ctx context.Context, id, submittedBy string) (models.Form, error)

	Withdraw(ctx context.Context, id, actorID string) (models.Form, error)

	Approve(ctx context.Context, id, approverID, note string) (models.Form, error)

	Refuse(ctx context.Context, id, approverID, note string) (models.Form, error)

	ListAwaitingApproval(ctx context.Context, limit int) ([]models.Form, error)

	ListApprovals(ctx context.Context, formID string) ([]models.Approval, error)

	Publish(ctx context.Context, id, candidateLinkKey, publishedBy string) (models.Form, models.PublicLink, error)

	Close(ctx context.Context, id, closedBy string) (models.Form, error)

	Delete(ctx context.Context, id string) error

	AddQuestion(ctx context.Context, in models.QuestionDraft) (models.Question, []models.QuestionOption, error)

	UpdateQuestion(ctx context.Context, formID, questionID string, in models.QuestionDraft) (models.Question, []models.QuestionOption, error)

	ListQuestions(ctx context.Context, formID string) ([]models.Question, error)

	ListOptions(ctx context.Context, questionID string) ([]models.QuestionOption, error)

	AddOption(ctx context.Context, questionID, label, actorID string) (models.AddOptionResult, error)

	VersionQuestion(ctx context.Context, questionID, actorID string, in models.QuestionDraft) (models.VersionQuestionResult, error)

	AddTarget(ctx context.Context, t models.Target) (models.Target, error)

	RemoveTarget(ctx context.Context, formID, targetID string) error

	ListTargets(ctx context.Context, formID string) ([]models.Target, error)

	Rollup(ctx context.Context, formID string) (models.Rollup, error)

	PickUp(ctx context.Context, formID, chapterID string) (models.Propagation, error)

	ListPropagation(ctx context.Context, formID string) ([]models.Propagation, error)

	ResolveLinkKey(ctx context.Context, key string) (models.PublicLink, models.Form, bool, error)

	UpsertRespondent(ctx context.Context, r models.Respondent) (models.Respondent, error)

	SubmitAnswers(ctx context.Context, formID, memberID, chapterID string, answers []models.Answer) ([]models.Answer, error)

	ListAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error)

	Declare(ctx context.Context, d models.Declaration) (models.Declaration, error)

	OpenPublicLink(ctx context.Context, key string) (models.ResolvedLink, error)

	RegisterRespondent(ctx context.Context, key, publicKey string) (models.RespondentPublicView, error)

	DeclareChapter(ctx context.Context, key, respondentID, declaredChapterID, declaredText string) (models.Declaration, error)
}

type formManager struct {
	db *gorm.DB
	st *store
}

func NewFormManager(db *gorm.DB, s *spec.Spec) (FormManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewFormManager: db must not be nil")
	}
	st, err := newStore(db, s, map[string]any{
		roleForm:           models.Form{},
		roleQuestion:       models.Question{},
		roleQuestionOption: models.QuestionOption{},
		roleTarget:         models.Target{},
		roleApproval:       models.Approval{},
		rolePropagation:    models.Propagation{},
		rolePublicLink:     models.PublicLink{},
		roleRespondent:     models.Respondent{},
		roleDeclaration:    models.Declaration{},
		roleAnswer:         models.Answer{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewFormManager: %w", err)
	}
	return &formManager{db: db, st: st}, nil
}

func (m *formManager) q(ctx context.Context, role string) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.st.Table(role))
}

func (m *formManager) table(role string) string { return m.st.Table(role) }

func (m *formManager) take(q *gorm.DB, role string, out any, notFound error) error {
	var rows []map[string]any
	if err := q.Limit(1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return notFound
	}
	return decode(m.st.Object(role), rows[0], out)
}

func (m *formManager) find(ctx context.Context, role, id string, out any, notFound error) error {
	return m.take(m.q(ctx, role).Where("id = ?", id), role, out, notFound)
}

func listOf[T any](m *formManager, q *gorm.DB, role string) ([]T, error) {
	var rows []map[string]any
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	o := m.st.Object(role)
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		var v T
		if err := decode(o, r, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (m *formManager) insert(ctx context.Context, role string, v any) error {
	return m.insertTx(m.db.WithContext(ctx), role, v)
}

func (m *formManager) insertTx(tx *gorm.DB, role string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return tx.Table(m.table(role)).Create(row).Error
}

func (m *formManager) replace(ctx context.Context, role, id string, v any) error {
	return m.replaceTx(m.db.WithContext(ctx), role, id, v)
}

func (m *formManager) replaceTx(tx *gorm.DB, role, id string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return tx.Table(m.table(role)).Where("id = ?", id).Updates(row).Error
}
