package mwanachamaforms

import "errors"

// Lookup and reference errors.
var (
	// ErrFormNotFound is returned when a Form id has no record.
	ErrFormNotFound = errors.New("mwanachamaforms: form not found")
	// ErrQuestionNotFound is returned when a Question id has no record, or
	// names a question belonging to a different form than the one addressed.
	ErrQuestionNotFound = errors.New("mwanachamaforms: question not found")
	// ErrTargetNotFound is returned when a Target id has no record, or names
	// a target belonging to a different form than the one addressed.
	ErrTargetNotFound = errors.New("mwanachamaforms: target not found")
	// ErrInvalidReference is returned when a write names an id (form_id,
	// question_id, ...) that does not exist.
	ErrInvalidReference = errors.New("mwanachamaforms: invalid reference")
	// ErrDuplicateTarget is returned when AddTarget names a (form, chapter)
	// pair that is already targeted.
	ErrDuplicateTarget = errors.New("mwanachamaforms: chapter is already targeted")
)

// Form-frame validation errors — Create/Update.
var (
	ErrMissingTitle     = errors.New("mwanachamaforms: title is required")
	ErrMissingClosesAt  = errors.New("mwanachamaforms: closes_at is required")
	ErrClosesInPast     = errors.New("mwanachamaforms: closes_at must be in the future")
	ErrWindowInvalid    = errors.New("mwanachamaforms: opens_at must be before closes_at")
	ErrAudienceConflict = errors.New("mwanachamaforms: a member-audience form cannot use interviewer collection")
)

// Lifecycle-transition errors.
var (
	// ErrNotDraft is returned when a draft-only operation (Update, Submit,
	// AddQuestion, UpdateQuestion, AddTarget, RemoveTarget) is attempted on
	// a form that has left StatusDraft.
	ErrNotDraft = errors.New("mwanachamaforms: form is not a draft")
	// ErrNotWithdrawable is returned when Withdraw is attempted on a form
	// that is neither submitted nor approved.
	ErrNotWithdrawable = errors.New("mwanachamaforms: form cannot be withdrawn from its current status")
	// ErrNotSubmitted is returned when Approve/Refuse is attempted on a
	// form that is not awaiting a decision.
	ErrNotSubmitted = errors.New("mwanachamaforms: form is not awaiting approval")
	// ErrMissingNote is returned when Refuse is called with a blank note.
	ErrMissingNote = errors.New("mwanachamaforms: a note is required to refuse")
	// ErrAwaitingApproval is returned when Publish is attempted on a form
	// that has been submitted but not yet decided.
	ErrAwaitingApproval = errors.New("mwanachamaforms: form is awaiting approval")
	// ErrNotOpen is returned when Close is attempted on a form that is not
	// open.
	ErrNotOpen = errors.New("mwanachamaforms: form is not open")
	// ErrPublished is returned when Delete is attempted on a form that has
	// been published (open or closed) — see models.Deletable.
	ErrPublished = errors.New("mwanachamaforms: a published form cannot be deleted")
	// ErrPublicFormNoTargets is returned when AddTarget is attempted on a
	// public-audience form.
	ErrPublicFormNoTargets = errors.New("mwanachamaforms: a public-audience form cannot carry targets")
)

// Question errors.
var (
	ErrMissingPrompt      = errors.New("mwanachamaforms: prompt is required")
	ErrMissingOptionLabel = errors.New("mwanachamaforms: every option label is required")
	// ErrFormNotPublished is returned when VersionQuestion is attempted on
	// a question whose form is still a draft — drafts use UpdateQuestion
	// instead.
	ErrFormNotPublished = errors.New("mwanachamaforms: form has not been published")
	// ErrNotCurrentVersion is returned when VersionQuestion is attempted on
	// a question that has already been superseded.
	ErrNotCurrentVersion = errors.New("mwanachamaforms: question is not the current version")
)

// Answering errors.
var (
	// ErrFormNotOpen is returned when SubmitAnswers is attempted on a form
	// that is not open.
	ErrFormNotOpen = errors.New("mwanachamaforms: form is not open")
	// ErrQuestionNotOnForm is returned when an answer names a question that
	// does not belong to the form being answered.
	ErrQuestionNotOnForm = errors.New("mwanachamaforms: question does not belong to this form")
	// ErrInvalidAnswer is returned when an answer's value does not match its
	// question's answer_type shape — see models.ValidateAnswer.
	ErrInvalidAnswer = errors.New("mwanachamaforms: invalid answer")
)
