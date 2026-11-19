package models

type Submission struct {
	ID               string           `json:"id"`
	FormID           string           `json:"form_id"`
	MemberID         string           `json:"member_id"`
	ChapterID        string           `json:"chapter_id,omitempty"`
	RecordRowID      string           `json:"record_row_id,omitempty"`
	PeriodKey        string           `json:"period_key,omitempty"`
	SubmittedAt      string           `json:"submitted_at"`
	EditedAt         string           `json:"edited_at,omitempty"`
	Collector        string           `json:"collector,omitempty"`
	CollectedAt      string           `json:"collected_at,omitempty"`
	Channel          string           `json:"channel,omitempty"`
	Language         string           `json:"language,omitempty"`
	PaidAt           string           `json:"paid_at,omitempty"`
	PaidBy           string           `json:"paid_by,omitempty"`
	RepeatOf         string           `json:"repeat_of,omitempty"`
	WithdrawnAt      string           `json:"withdrawn_at,omitempty"`
	Decision         Decision         `json:"decision,omitempty"`
	DecidedBy        string           `json:"decided_by,omitempty"`
	DecidedAt        string           `json:"decided_at,omitempty"`
	DecisionNote     string           `json:"decision_note,omitempty"`
	CertificateRef   string           `json:"certificate_ref,omitempty"`
	ValidUntil       string           `json:"valid_until,omitempty"`
	InspectionResult InspectionResult `json:"inspection_result,omitempty"`
	InspectionNote   string           `json:"inspection_note,omitempty"`
	InspectedBy      string           `json:"inspected_by,omitempty"`
	InspectedAt      string           `json:"inspected_at,omitempty"`
	VoiceKey         string           `json:"-"`
	Paid             bool             `json:"paid" spec:"-"`
	Answers          int              `json:"answers" spec:"-"`
	FormTitle        string           `json:"form_title,omitempty" spec:"-"`
}

const (
	ChannelApp   = "app"
	ChannelLink  = "link"
	ChannelField = "field"
	ChannelPaper = "paper"
)
