package models

// PropagationKind is a chapter's progress reaching a member-audience Form
// materialized by Publish. Only Targeted and PickedUp are ever written by
// any method in this package — Localized and Pushed are carried for schema
// parity with the gateway (whose own domain never writes them either; see
// this repo's CLAUDE.md), not live behavior.
type PropagationKind string

const (
	PropagationTargeted  PropagationKind = "targeted"
	PropagationPickedUp  PropagationKind = "picked_up"
	PropagationLocalized PropagationKind = "localized"
	PropagationPushed    PropagationKind = "pushed"
)

// Propagation is one Target's row, materialized when its Form is published
// (Publish creates one Propagation per Target, State=Targeted) and updated
// by PickUp (State=PickedUp). LastReminderAt is carried for the same schema
// parity as Localized/Pushed above — nothing writes it.
type Propagation struct {
	ID                string          `json:"id"`
	FormID            string          `json:"form_id"`
	ChapterID         string          `json:"chapter_id"`
	State             PropagationKind `json:"state"`
	TargetedAt        string          `json:"targeted_at"`
	PickedUpAt        string          `json:"picked_up_at,omitempty"`
	PushedAt          string          `json:"pushed_at,omitempty"`
	PushedByChapterID string          `json:"pushed_by_chapter_id,omitempty"`
	LastReminderAt    string          `json:"last_reminder_at,omitempty"`
}

// Rollup summarizes a Form's Propagation rows. Stalled is exactly "targeted
// but PickedUpAt is still empty" — no time-based staleness threshold,
// mirroring the gateway's survey.Rollup computation exactly.
type Rollup struct {
	Targeted          int      `json:"targeted"`
	PickedUp          int      `json:"picked_up"`
	Stalled           int      `json:"stalled"`
	StalledChapterIDs []string `json:"stalled_chapter_ids,omitempty"`
}
