package models

import (
	"fmt"
	"time"
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

func ValidateAudienceCollection(a Audience, m CollectionMode) error {
	if a == AudienceMember && m == CollectionInterviewer {
		return fmt.Errorf("a member-audience form cannot use interviewer collection")
	}
	return nil
}
