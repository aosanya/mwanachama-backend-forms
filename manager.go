package mwanachamaforms

import (
	"context"
	"fmt"
	"sync/atomic"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/identity"
	"github.com/aosanya/mwanachama-backend-shared/spec"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type (
	Form                  = models.Form
	Status                = models.Status
	Audience              = models.Audience
	CollectionMode        = models.CollectionMode
	Landing               = models.Landing
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
	WaitingForm           = models.WaitingForm
	AnsweredForm          = models.AnsweredForm
	Approval              = models.Approval
	Decision              = models.Decision
	PublicLink            = models.PublicLink
	LinkStatus            = models.LinkStatus
	Respondent            = models.Respondent
	Declaration           = models.Declaration
	DeclarationCount      = models.DeclarationCount
	Answer                = models.Answer
	Submission            = models.Submission
	Send                  = models.Send
	ResultReport          = models.ResultReport
	QuestionDraft         = models.QuestionDraft
	ResolvedLink          = models.ResolvedLink
	LinkedQuestion        = models.LinkedQuestion
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
	LandingNone   = models.LandingNone
	LandingThread = models.LandingThread
	LandingCase   = models.LandingCase
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

func ValidateAnswerWithin(q Question, options []QuestionOption, a Answer, defaultMaxFiles int) error {
	return models.ValidateAnswerWithin(q, options, a, defaultMaxFiles)
}

func ValidateValidators(q Question) error {
	return models.ValidateValidators(q)
}

func ValidateWindow(opensAt, closesAt string) error {
	return models.ValidateWindow(opensAt, closesAt)
}

func ValidateAudienceCollection(a Audience, m CollectionMode) error {
	return models.ValidateAudienceCollection(a, m)
}

func ValidateLanding(standing bool, l Landing) error {
	return models.ValidateLanding(standing, l)
}

func VisibleToMembers(s Status) bool { return models.VisibleToMembers(s) }

func Deletable(s Status) bool { return models.Deletable(s) }

type FormManager interface {
	identity.Owner
	Translator
	TranslatedReader

	Create(ctx context.Context, f models.Form) (models.Form, error)

	Get(ctx context.Context, id string) (models.Form, error)

	Update(ctx context.Context, f models.Form) (models.Form, error)

	ListForChapter(ctx context.Context, chapterID string) ([]models.Form, error)

	ListReachable(ctx context.Context, ancestry []string) ([]models.Form, error)

	Register(ctx context.Context, q models.RegisterQuery) (models.RegisterPage, error)

	ListWaiting(ctx context.Context, callerID string) ([]models.WaitingForm, error)

	ListAnswered(ctx context.Context, callerID string) ([]models.AnsweredForm, error)

	UseGroups(g Groups)

	UseStandingAudience(a StandingAudience)

	UseNames(n Names)

	UseMembers(mb Members)

	ListMissed(ctx context.Context, formID, periodKey string) ([]models.MissedMember, error)

	UseDeclarable(d Declarable)

	UseApprovalRequired(required bool)
	FormTypes() []models.FormType
	UseDefaultMaxFiles(n int)

	Submit(ctx context.Context, id, submittedBy string) (models.Form, error)

	Withdraw(ctx context.Context, id, actorID string) (models.Form, error)

	Approve(ctx context.Context, id, approverID, note string) (models.Form, error)
	ApproveSlot(ctx context.Context, id, approverID, slot, note string) (models.Form, error)

	Refuse(ctx context.Context, id, approverID, note string) (models.Form, error)

	ListAwaitingApproval(ctx context.Context, limit int) ([]models.Form, error)

	ListApprovals(ctx context.Context, formID string) ([]models.Approval, error)

	Publish(ctx context.Context, id, candidateLinkKey, publishedBy, publishingAuthority string) (models.Form, models.PublicLink, error)

	Close(ctx context.Context, id, closedBy string) (models.Form, error)

	Delete(ctx context.Context, id string) error

	PublishResults(ctx context.Context, formID, gitPath, gitCommit string, chapterIDs []string, publishedBy string) (models.ResultReport, error)

	LatestResults(ctx context.Context, formID, callerID string) (models.ResultReport, error)

	AddQuestion(ctx context.Context, in models.QuestionDraft) (models.Question, []models.QuestionOption, error)

	UpdateQuestion(ctx context.Context, formID, questionID string, in models.QuestionDraft) (models.Question, []models.QuestionOption, error)

	RemoveQuestion(ctx context.Context, formID, questionID string) error

	MoveQuestion(ctx context.Context, formID, questionID string, position int) ([]models.Question, error)

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

	SubmitAnswers(ctx context.Context, formID, memberID string, answers []models.Answer) ([]models.Answer, error)
	SubmitAnswersAs(ctx context.Context, formID, memberID, answeringAs string, answers []models.Answer) ([]models.Answer, error)
	SubmitAnswersIn(ctx context.Context, formID, memberID, answeringAs, language string, answers []models.Answer) ([]models.Answer, error)
	SubmitAnswersFor(ctx context.Context, formID, memberID, answeringAs, language, periodKey string, answers []models.Answer) ([]models.Answer, error)

	ListAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error)
	ListPreviousAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error)

	ListAllAnswers(ctx context.Context, formID, memberID, periodKey string) ([]models.Answer, error)

	GetSubmission(ctx context.Context, formID, memberID string) (models.Submission, error)

	ListSubmissions(ctx context.Context, formID, memberID, periodKey string) ([]models.Submission, error)

	MarkPaid(ctx context.Context, formID, memberID string, paid bool, markedBy string) (models.Submission, error)

	GetSend(ctx context.Context, formID, sendID, memberID string) (models.Send, error)

	Declare(ctx context.Context, d models.Declaration) (models.Declaration, error)

	OpenPublicLink(ctx context.Context, key string) (models.ResolvedLink, error)

	PublicLinkQuestions(ctx context.Context, key string) ([]models.LinkedQuestion, error)

	RegisterRespondent(ctx context.Context, key, publicKey string) (models.RespondentPublicView, error)

	DeclareChapter(ctx context.Context, key, respondentID, declaredChapterID, declaredText string) (models.Declaration, error)

	ClaimRespondent(ctx context.Context, key, publicKey, memberID string) (models.RespondentPublicView, error)

	ListDeclarations(ctx context.Context, formID string) ([]models.Declaration, error)

	ForRun(runID string) (FormManager, error)

	CountDeclarations(ctx context.Context, formID string) ([]models.DeclarationCount, error)
}

type formManager struct {
	db     *gorm.DB
	st     *store
	groups Groups
	names  Names

	standingAudience StandingAudience

	members Members

	declarable Declarable

	approvalRequired atomic.Bool
	defaultMaxFiles  int
	formTypes        []models.FormType
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
		roleClaim:          models.Claim{},
		roleAnswer:         models.Answer{},
		roleSubmission:     models.Submission{},
		roleResultReport:   models.ResultReport{},
		roleTranslation:    models.Translation{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewFormManager: %w", err)
	}
	settings, err := resolveSettings(db, s)
	if err != nil {
		return nil, fmt.Errorf("NewFormManager: %w", err)
	}
	m := &formManager{db: db, st: st, formTypes: settings.formTypes}
	m.approvalRequired.Store(settings.approvalRequired)
	return m, nil
}

func (m *formManager) q(ctx context.Context, role string) *gorm.DB {
	return m.st.Query(ctx, role)
}

func (m *formManager) from(tx *gorm.DB, role string) *gorm.DB {
	q := tx.Table(m.table(role))
	if id := m.st.RunID(); id != "" {
		q = q.Where(spec.RunIDColumn+" = ?", id)
	}
	return q
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
	row, err := m.st.Row(role, v)
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
	return m.from(tx, role).Where("id = ?", id).Updates(row).Error
}
