package models

type Submission struct {
	ID          string `json:"id"`
	FormID      string `json:"form_id"`
	MemberID    string `json:"member_id"`
	ChapterID   string `json:"chapter_id,omitempty"`
	PeriodKey   string `json:"period_key,omitempty"`
	SubmittedAt string `json:"submitted_at"`
	EditedAt    string `json:"edited_at,omitempty"`
	Collector   string `json:"collector,omitempty"`
	CollectedAt string `json:"collected_at,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Language    string `json:"language,omitempty"`
	PaidAt      string `json:"paid_at,omitempty"`
	PaidBy      string `json:"paid_by,omitempty"`
	Paid        bool   `json:"paid" spec:"-"`
	Answers     int    `json:"answers" spec:"-"`
}

const (
	ChannelApp   = "app"
	ChannelLink  = "link"
	ChannelField = "field"
	ChannelPaper = "paper"
)
