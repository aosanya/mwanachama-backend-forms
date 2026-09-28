package mwanachamaforms

import "errors"

var (
	ErrFormNotFound     = errors.New("mwanachamaforms: form not found")
	ErrQuestionNotFound = errors.New("mwanachamaforms: question not found")
	ErrTargetNotFound   = errors.New("mwanachamaforms: target not found")
	ErrInvalidReference = errors.New("mwanachamaforms: invalid reference")
	ErrDuplicateTarget  = errors.New("mwanachamaforms: chapter is already targeted")
	ErrLinkKeyTaken     = errors.New("mwanachamaforms: public link key is already taken")
	ErrLinkNotFound     = errors.New("not found")
)

var (
	ErrInvalidForm     = errors.New("mwanachamaforms: invalid form")
	ErrInvalidQuestion = errors.New("mwanachamaforms: invalid question")
	ErrInvalidAnswer   = errors.New("mwanachamaforms: invalid answer")
)

var (
	ErrMissingTitle     = errors.New("mwanachamaforms: title is required")
	ErrMissingClosesAt  = errors.New("mwanachamaforms: closes_at is required")
	ErrClosesInPast     = errors.New("mwanachamaforms: closes_at must be in the future")
	ErrWindowInvalid    = errors.New("mwanachamaforms: opens_at must be before closes_at")
	ErrAudienceConflict = errors.New("mwanachamaforms: a member-audience form cannot use interviewer collection")
)

var (
	ErrNotDraft            = errors.New("mwanachamaforms: form is not a draft")
	ErrNotWithdrawable     = errors.New("mwanachamaforms: form cannot be withdrawn from its current status")
	ErrNotSubmitted        = errors.New("mwanachamaforms: form is not awaiting approval")
	ErrMissingNote         = errors.New("mwanachamaforms: a note is required to refuse")
	ErrAwaitingApproval    = errors.New("mwanachamaforms: form is awaiting approval")
	ErrNotOpen             = errors.New("mwanachamaforms: form is not open")
	ErrPublished           = errors.New("mwanachamaforms: a published form cannot be deleted")
	ErrPublicFormNoTargets = errors.New("mwanachamaforms: a public-audience form cannot carry targets")
)

var (
	ErrMissingPrompt      = errors.New("mwanachamaforms: prompt is required")
	ErrMissingOptionLabel = errors.New("mwanachamaforms: every option label is required")
	ErrFormNotPublished   = errors.New("mwanachamaforms: form has not been published")
	ErrNotCurrentVersion  = errors.New("mwanachamaforms: question is not the current version")
)

var (
	ErrFormNotOpen       = errors.New("mwanachamaforms: form is not open")
	ErrQuestionNotOnForm = errors.New("mwanachamaforms: question does not belong to this form")
)
