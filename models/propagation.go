package models

type PropagationKind string

const (
	PropagationTargeted  PropagationKind = "targeted"
	PropagationPickedUp  PropagationKind = "picked_up"
	PropagationLocalized PropagationKind = "localized"
	PropagationPushed    PropagationKind = "pushed"
)

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

type Rollup struct {
	Targeted          int      `json:"targeted"`
	PickedUp          int      `json:"picked_up"`
	Stalled           int      `json:"stalled"`
	StalledChapterIDs []string `json:"stalled_chapter_ids,omitempty"`
}
