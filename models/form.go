package models

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusApproved  Status = "approved"
	StatusOpen      Status = "open"
	StatusClosed    Status = "closed"
)

type Audience string

const (
	AudienceMember Audience = "member"
	AudiencePublic Audience = "public"
)

type CollectionMode string

const (
	CollectionSelf        CollectionMode = "self"
	CollectionInterviewer CollectionMode = "interviewer"
)

type Form struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	OriginatorChapterID string `json:"originator_chapter_id"`
	Status              Status `json:"status"`

	OpensAt            string   `json:"opens_at,omitempty"`
	ClosesAt           string   `json:"closes_at"`
	Language           string   `json:"language,omitempty"`
	WindowAdvisory     bool     `json:"window_advisory"`
	Standing           bool     `json:"standing"`
	PublishedAt        string   `json:"published_at,omitempty"`
	PublishedBy        string   `json:"published_by,omitempty"`
	ClosedAt           string   `json:"closed_at,omitempty"`
	ClosedBy           string   `json:"closed_by,omitempty"`
	SubmittedAt        string   `json:"submitted_at,omitempty"`
	SubmittedBy        string   `json:"submitted_by,omitempty"`
	ApprovedAt         string   `json:"approved_at,omitempty"`
	ApprovedBy         string   `json:"approved_by,omitempty"`
	ApprovalRoles      []string `json:"approval_roles,omitempty"`
	ResultsPublishedAt string   `json:"results_published_at,omitempty"`

	PublisherName       string `json:"publisher_name,omitempty"`
	PublishingAuthority string `json:"publishing_authority,omitempty"`

	CreatedBy      string         `json:"created_by"`
	Audience       Audience       `json:"audience"`
	CollectionMode CollectionMode `json:"collection_mode"`
	Schedule       FormSchedule   `json:"schedule,omitzero"`

	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
}

func Deletable(s Status) bool {
	return !VisibleToMembers(s)
}

func VisibleToMembers(s Status) bool {
	return s == StatusOpen || s == StatusClosed
}

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

func (f Form) OutsideWindow(now time.Time) bool {
	if f.WindowAdvisory {
		return false
	}
	if opens, err := time.Parse(time.RFC3339Nano, f.OpensAt); err == nil && now.Before(opens) {
		return true
	}
	if closes, err := time.Parse(time.RFC3339Nano, f.ClosesAt); err == nil && now.After(closes) {
		return true
	}
	return false
}

func ValidateAudienceCollection(a Audience, m CollectionMode) error {
	if a == AudienceMember && m == CollectionInterviewer {
		return fmt.Errorf("a member-audience form cannot use interviewer collection")
	}
	return nil
}

const MaxPublishingAuthority = 120

const MaxPublisherName = 120

func ValidatePublishingAuthority(a string) error {
	if utf8.RuneCountInString(a) > MaxPublishingAuthority {
		return fmt.Errorf("publishing_authority is longer than %d characters", MaxPublishingAuthority)
	}
	for _, r := range a {
		if unicode.IsControl(r) {
			return fmt.Errorf("publishing_authority must be one line of text")
		}
	}
	return nil
}

func ShortenPublisherName(name string) string {
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, name))
	runes := []rune(name)
	if len(runes) > MaxPublisherName {
		return strings.TrimSpace(string(runes[:MaxPublisherName]))
	}
	return name
}
