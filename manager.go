package mwanachamaforms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

// Form, Question, Target, ... are aliases of their models. counterparts,
// and the constants and functions below forward to their models. namesakes,
// so a caller needs only this package's import, never models's directly.
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
	AnswerType            = models.AnswerType
)

// Status values.
const (
	StatusDraft     = models.StatusDraft
	StatusSubmitted = models.StatusSubmitted
	StatusApproved  = models.StatusApproved
	StatusOpen      = models.StatusOpen
	StatusClosed    = models.StatusClosed
)

// Audience values.
const (
	AudienceMember = models.AudienceMember
	AudiencePublic = models.AudiencePublic
)

// CollectionMode values.
const (
	CollectionSelf        = models.CollectionSelf
	CollectionInterviewer = models.CollectionInterviewer
)

// Decision values.
const (
	DecisionApproved = models.DecisionApproved
	DecisionRefused  = models.DecisionRefused
)

// AnswerType values in active use — see [models.AnswerType]'s doc for the
// full schema-parity set.
const (
	AnswerYesNo        = models.AnswerYesNo
	AnswerSingleChoice = models.AnswerSingleChoice
	AnswerFreeText     = models.AnswerFreeText
)

// LinkStatus values.
const (
	LinkActive = models.LinkActive
)

// PropagationKind values ever written by this package — see
// [models.PropagationKind]'s doc for the full schema-parity set.
const (
	PropagationTargeted = models.PropagationTargeted
	PropagationPickedUp = models.PropagationPickedUp
)

// TimeLayout forwards to [models.TimeLayout].
const TimeLayout = models.TimeLayout

// NowRFC3339 forwards to [models.NowRFC3339].
func NowRFC3339() string { return models.NowRFC3339() }

// ValidateAnswer forwards to [models.ValidateAnswer].
func ValidateAnswer(q Question, options []QuestionOption, a Answer) error {
	return models.ValidateAnswer(q, options, a)
}

// ValidateWindow forwards to [models.ValidateWindow].
func ValidateWindow(opensAt, closesAt string) error {
	return models.ValidateWindow(opensAt, closesAt)
}

// ValidateAudienceCollection forwards to [models.ValidateAudienceCollection].
func ValidateAudienceCollection(a Audience, m CollectionMode) error {
	return models.ValidateAudienceCollection(a, m)
}

// VisibleToMembers forwards to [models.VisibleToMembers].
func VisibleToMembers(s Status) bool { return models.VisibleToMembers(s) }

// Deletable forwards to [models.Deletable].
func Deletable(s Status) bool { return models.Deletable(s) }

// FormManager is the primary interface for Form lifecycle management —
// compose, approve, publish, answer, close — and every type nested under a
// Form (Question, QuestionOption, Target, Approval, Propagation,
// PublicLink, Respondent, Declaration, Answer). This is a single-tenant
// package — one deployment serves one agency, so no method takes a
// tenant-scoping argument, mirroring
// mwanachama-backend-actor.UserManager's shape.
//
// Implementations must be safe for concurrent use.
type FormManager interface {
	// Create creates a new Form in [models.StatusDraft]. Validates Title,
	// ClosesAt, the opens/closes window ([models.ValidateWindow]) and the
	// audience/collection pairing ([models.ValidateAudienceCollection]).
	Create(ctx context.Context, f models.Form) (models.Form, error)
	// Get retrieves a single Form by id. Returns [ErrFormNotFound] if none
	// exists.
	Get(ctx context.Context, id string) (models.Form, error)
	// Update rewrites a draft Form's Title/ClosesAt/OpensAt/Audience/
	// CollectionMode. Returns [ErrNotDraft] once the form has left
	// [models.StatusDraft].
	Update(ctx context.Context, f models.Form) (models.Form, error)
	// ListForChapter returns every Form (any status) originated by a
	// chapter, created_at-then-id order.
	ListForChapter(ctx context.Context, chapterID string) ([]models.Form, error)
	// ListReachable returns every member-visible Form ([models.VisibleToMembers])
	// whose Target set reaches ancestry — ancestry[0] is the caller's own
	// chapter, the rest are its ancestors up to the root. A Target on the
	// caller's own chapter always reaches; a Target on an ancestor reaches
	// only when it carries IncludesDescendants.
	ListReachable(ctx context.Context, ancestry []string) ([]models.Form, error)
	// Register returns a filtered, paginated page of Forms with their
	// question/respondent counts attached — see [models.RegisterQuery]'s doc.
	Register(ctx context.Context, q models.RegisterQuery) (models.RegisterPage, error)

	// Submit moves a draft Form to [models.StatusSubmitted] and opens a new
	// [models.Approval] awaiting a decision. Returns [ErrNotDraft] otherwise.
	Submit(ctx context.Context, id, submittedBy string) (models.Form, error)
	// Withdraw reverts a submitted or approved Form to
	// [models.StatusDraft], closing its open approval as refused (note
	// "withdrawn by its author before a decision"). Returns
	// [ErrNotWithdrawable] otherwise.
	Withdraw(ctx context.Context, id, actorID string) (models.Form, error)
	// Approve decides a submitted Form's open approval as
	// [models.DecisionApproved], moving it to [models.StatusApproved].
	// Returns [ErrNotSubmitted] otherwise.
	Approve(ctx context.Context, id, approverID, note string) (models.Form, error)
	// Refuse decides a submitted Form's open approval as
	// [models.DecisionRefused], reverting it to [models.StatusDraft].
	// Returns [ErrMissingNote] for a blank note, [ErrNotSubmitted] otherwise.
	Refuse(ctx context.Context, id, approverID, note string) (models.Form, error)
	// ListAwaitingApproval returns every submitted Form, SubmittedAt-then-id
	// order, truncated to limit when limit > 0.
	ListAwaitingApproval(ctx context.Context, limit int) ([]models.Form, error)
	// ListApprovals returns every approval cycle a Form has been through,
	// newest first.
	ListApprovals(ctx context.Context, formID string) ([]models.Approval, error)

	// Publish opens a draft or approved Form ([models.StatusOpen]).
	// Returns [ErrAwaitingApproval] for a submitted form,
	// [ErrClosesInPast] if ClosesAt has already passed, [ErrNotDraft]
	// otherwise; a no-op returning the current state for an already-open
	// form. A public-audience form gets exactly one [models.PublicLink]
	// (Key set to candidateLinkKey verbatim — this package does not
	// generate or deduplicate the key itself); a member-audience form gets
	// one [models.Propagation] per existing Target instead.
	Publish(ctx context.Context, id, candidateLinkKey, publishedBy string) (models.Form, models.PublicLink, error)
	// Close closes an open Form ([models.StatusClosed]). Returns
	// [ErrNotOpen] otherwise; a no-op returning the current state for an
	// already-closed form.
	Close(ctx context.Context, id, closedBy string) (models.Form, error)
	// Delete removes a Form and its Questions/QuestionOptions/Targets/
	// Approvals. Returns [ErrPublished] once the form has been published
	// (open or closed) — see [models.Deletable].
	Delete(ctx context.Context, id string) error

	// AddQuestion appends a new Question (and its options) to a draft Form.
	// Ordinal and Version are always server-assigned, ignoring any caller
	// value. Returns [ErrNotDraft] once the form has left draft.
	AddQuestion(ctx context.Context, q models.Question, options []models.QuestionOption) (models.Question, []models.QuestionOption, error)
	// UpdateQuestion replaces a draft question's wording and options
	// wholesale (old options are deleted, not merged). Returns
	// [ErrNotDraft] once the form has left draft, [ErrQuestionNotFound] if
	// questionID does not belong to formID.
	UpdateQuestion(ctx context.Context, formID, questionID string, next models.Question, options []models.QuestionOption) (models.Question, []models.QuestionOption, error)
	// ListQuestions returns every version of every question on a form,
	// Ordinal order.
	ListQuestions(ctx context.Context, formID string) ([]models.Question, error)
	// ListOptions returns a question's options, Ordinal order.
	ListOptions(ctx context.Context, questionID string) ([]models.QuestionOption, error)
	// AddOption appends one option to an existing question — the one
	// additive content edit allowed regardless of the form's status.
	AddOption(ctx context.Context, questionID, label, actorID string) (models.AddOptionResult, error)
	// VersionQuestion supersedes a published question with a new row
	// (Ordinal and AnswerType carried over, wording/options replaced),
	// leaving the predecessor and its existing answers untouched. Returns
	// [ErrFormNotPublished] on a draft form, [ErrNotCurrentVersion] if
	// questionID has already been superseded.
	VersionQuestion(ctx context.Context, questionID string, next models.Question, options []models.QuestionOption, actorID string) (models.VersionQuestionResult, error)

	// AddTarget aims a draft, member-audience Form at a chapter. Returns
	// [ErrPublicFormNoTargets] for a public-audience form,
	// [ErrDuplicateTarget] for a chapter already targeted, [ErrNotDraft]
	// once the form has left draft.
	AddTarget(ctx context.Context, t models.Target) (models.Target, error)
	// RemoveTarget removes a draft form's target. Returns [ErrTargetNotFound]
	// if targetID does not belong to formID, [ErrNotDraft] otherwise.
	RemoveTarget(ctx context.Context, formID, targetID string) error
	// ListTargets returns a form's targets, created_at order.
	ListTargets(ctx context.Context, formID string) ([]models.Target, error)

	// Rollup summarizes a form's propagation: how many chapters were
	// targeted, picked up, or are still stalled (targeted with no pickup).
	Rollup(ctx context.Context, formID string) (models.Rollup, error)
	// PickUp records a chapter picking up a form it was targeted with.
	// Returns [ErrTargetNotFound] if the chapter was never targeted (Publish
	// must run first); a no-op returning the current state if already
	// picked up.
	PickUp(ctx context.Context, formID, chapterID string) (models.Propagation, error)
	// ListPropagation returns a form's propagation rows, TargetedAt order.
	ListPropagation(ctx context.Context, formID string) ([]models.Propagation, error)

	// ResolveLinkKey resolves a public link key to its link and form,
	// found=false (never an error) for a missing, retired, or otherwise
	// non-live link/form pair — a deliberately indistinguishable refusal.
	ResolveLinkKey(ctx context.Context, key string) (models.PublicLink, models.Form, bool, error)
	// UpsertRespondent looks a respondent up by PublicKey, creating one if
	// none exists. An existing respondent's other fields are never
	// overwritten by this call.
	UpsertRespondent(ctx context.Context, r models.Respondent) (models.Respondent, error)
	// SubmitAnswers validates and writes every answer in one all-or-nothing
	// batch, upserting on (QuestionID, MemberID) — a resubmit keeps the
	// original AnsweredAt/ChapterID and stamps EditedAt. Returns
	// [ErrFormNotOpen], [ErrQuestionNotOnForm], or [ErrInvalidAnswer].
	SubmitAnswers(ctx context.Context, formID, memberID, chapterID string, answers []models.Answer) ([]models.Answer, error)
	// ListAnswers returns a member's answers on a form, question-Ordinal order.
	ListAnswers(ctx context.Context, formID, memberID string) ([]models.Answer, error)
	// Declare records (overwriting any prior declaration) a respondent's
	// self-reported chapter for a form.
	Declare(ctx context.Context, d models.Declaration) (models.Declaration, error)
}

// formManager is the concrete implementation of [FormManager].
type formManager struct {
	db     *gorm.DB
	tables TableNames
}

// NewFormManager constructs a [FormManager] backed by db, reading and
// writing the tables named by t (see [DefaultTableNames]). Callers must run
// [Migrate] against the same db and t before use. Returns an error if db is
// nil.
func NewFormManager(db *gorm.DB, t TableNames) (FormManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewFormManager: db must not be nil")
	}
	return &formManager{db: db, tables: t}, nil
}
