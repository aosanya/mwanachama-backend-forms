package models

type WaitingForm struct {
	FormID              string   `json:"form_id"`
	Title               string   `json:"title"`
	OriginatorChapterID string   `json:"originator_chapter_id"`
	PublishedBy         string   `json:"published_by,omitempty"`
	PublisherName       string   `json:"publisher_name,omitempty"`
	PublishingAuthority string   `json:"publishing_authority,omitempty"`
	PublishedAt         string   `json:"published_at,omitempty"`
	OpensAt             string   `json:"opens_at,omitempty"`
	ClosesAt            string   `json:"closes_at"`
	Questions           int      `json:"questions"`
	ReachedVia          []string `json:"reached_via"`
	Answered            bool     `json:"answered"`
	AnsweredAs          []string `json:"answered_as"`
	UnansweredAs        []string `json:"unanswered_as"`
}
