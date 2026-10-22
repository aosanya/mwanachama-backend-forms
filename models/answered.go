package models

type AnsweredForm struct {
	FormID              string `json:"form_id"`
	Title               string `json:"title"`
	OriginatorChapterID string `json:"originator_chapter_id"`
	PublishedBy         string `json:"published_by,omitempty"`
	PublisherName       string `json:"publisher_name,omitempty"`
	PublishingAuthority string `json:"publishing_authority,omitempty"`
	PublishedAt         string `json:"published_at,omitempty"`
	OpensAt             string `json:"opens_at,omitempty"`
	ClosesAt            string `json:"closes_at"`
	ClosedAt            string `json:"closed_at,omitempty"`
	Status              Status `json:"status"`
	Open                bool   `json:"open"`
	ChapterID           string `json:"chapter_id,omitempty"`
	AnsweredAt          string `json:"answered_at"`
	EditedAt            string `json:"edited_at,omitempty"`
	PeriodKey           string `json:"period_key,omitempty"`
}
