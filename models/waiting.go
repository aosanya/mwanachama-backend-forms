package models

type WaitingForm struct {
	FormID              string   `json:"form_id"`
	Title               string   `json:"title"`
	OriginatorChapterID string   `json:"originator_chapter_id"`
	PublishedBy         string   `json:"published_by,omitempty"`
	PublishedAt         string   `json:"published_at,omitempty"`
	OpensAt             string   `json:"opens_at,omitempty"`
	ClosesAt            string   `json:"closes_at"`
	Questions           int      `json:"questions"`
	ReachedVia          []string `json:"reached_via"`
	Answered            bool     `json:"answered"`
}
