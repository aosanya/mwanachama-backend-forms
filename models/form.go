package models

import (
	"fmt"
	"time"
)

// Status is a Form's lifecycle stage. Mirrors
// mwanachama-backend-api-gateway's internal/domain/survey.Status field for
// field — this package's own naming-consistency pass renames the gateway's
// root noun Survey to Form (this repo's own name), not this enum's values.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusApproved  Status = "approved"
	StatusOpen      Status = "open"
	StatusClosed    Status = "closed"
)

// Audience is who a Form is aimed at.
type Audience string

const (
	AudienceMember Audience = "member"
	AudiencePublic Audience = "public"
)

// CollectionMode is how a Form's answers get captured.
type CollectionMode string

const (
	CollectionSelf        CollectionMode = "self"
	CollectionInterviewer CollectionMode = "interviewer"
)

// Form is a survey/questionnaire — the gateway's internal/domain/survey.Survey,
// renamed here (this repo's own naming-consistency pass, mirroring
// mwanachama-backend-actor's Member->Actor/Chapter->Group). Every SurveyID
// field the gateway carries on Form's related types becomes FormID here.
//
// OpensAt/PublishedAt/PublishedBy/ClosedAt/ClosedBy/SubmittedAt/SubmittedBy/
// ApprovedAt/ApprovedBy/ResultsPublishedAt are plain strings with "" meaning
// unset, mirroring this package's existing convention for an optional string
// field (see Group.ParentID in mwanachama-backend-actor) rather than a
// pointer — a nil *time.Time in the gateway's struct becomes an empty string
// here, and every timestamp that is set is a [TimeLayout] string, not a
// pointer to time.Time (see time.go's doc for why).
//
// CreatedAt/LastUpdated do not exist on the gateway's Survey — it relies on
// its Postgres sequential id ("survey-1", "survey-2", ...) for chronological
// list order. This repo mints ids as UUIDs (mwanachama-backend-actor's own
// storage convention), which carry no such order, so both fields are added
// here purely to give ListForChapter a stable, meaningful sort — the same
// reasoning mwanachama-backend-actor already applied to Group and
// ActorGroupAssignment when it made the identical UUID switch.
type Form struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	OriginatorChapterID string `json:"originator_chapter_id"`
	Status              Status `json:"status"`

	OpensAt            string `json:"opens_at,omitempty"`
	ClosesAt           string `json:"closes_at"`
	PublishedAt        string `json:"published_at,omitempty"`
	PublishedBy        string `json:"published_by,omitempty"`
	ClosedAt           string `json:"closed_at,omitempty"`
	ClosedBy           string `json:"closed_by,omitempty"`
	SubmittedAt        string `json:"submitted_at,omitempty"`
	SubmittedBy        string `json:"submitted_by,omitempty"`
	ApprovedAt         string `json:"approved_at,omitempty"`
	ApprovedBy         string `json:"approved_by,omitempty"`
	ResultsPublishedAt string `json:"results_published_at,omitempty"`

	CreatedBy      string         `json:"created_by"`
	Audience       Audience       `json:"audience"`
	CollectionMode CollectionMode `json:"collection_mode"`

	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
}

// Deletable reports whether a Form in this status may be deleted outright.
func Deletable(s Status) bool {
	return !VisibleToMembers(s)
}

// VisibleToMembers reports whether a Form in this status may be seen by the
// members it targets — open and closed only, matching
// mwanachama-backend-api-gateway's survey.VisibleToMembers exactly.
func VisibleToMembers(s Status) bool {
	return s == StatusOpen || s == StatusClosed
}

// ValidateWindow enforces that opensAt, when set, is strictly before
// closesAt — mirrors the gateway's survey_window_runs_forwards (DEV-314). An
// empty opensAt means "on publish" and is always valid regardless of
// closesAt. Returns a plain error, not a package sentinel — callers (this
// repo's CreateForm/UpdateForm) wrap it with [ErrWindowInvalid], the same
// division of labor as models.ValidateAttributes / [ErrInvalidActor] in
// mwanachama-backend-actor.
func ValidateWindow(opensAt, closesAt string) error {
	if closesAt == "" {
		return nil
	}
	ct, err := time.Parse(time.RFC3339Nano, closesAt)
	if err != nil {
		return fmt.Errorf("closes_at: %w", err)
	}
	if opensAt == "" {
		return nil
	}
	ot, err := time.Parse(time.RFC3339Nano, opensAt)
	if err != nil {
		return fmt.Errorf("opens_at: %w", err)
	}
	if !ct.After(ot) {
		return fmt.Errorf("closes_at must be after opens_at")
	}
	return nil
}

// ValidateAudienceCollection enforces that a member-audience Form never uses
// interviewer collection — mirrors the gateway's
// survey_member_is_never_interviewed. Returns a plain error; callers wrap it
// with [ErrAudienceConflict].
func ValidateAudienceCollection(a Audience, m CollectionMode) error {
	if a == AudienceMember && m == CollectionInterviewer {
		return fmt.Errorf("a member-audience form cannot use interviewer collection")
	}
	return nil
}
