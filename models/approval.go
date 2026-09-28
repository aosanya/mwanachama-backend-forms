package models

type Decision string

const (
	DecisionApproved Decision = "approved"
	DecisionRefused  Decision = "refused"
)

type Approval struct {
	ID          string   `json:"id"`
	FormID      string   `json:"form_id"`
	RequestedBy string   `json:"requested_by"`
	RequestedAt string   `json:"requested_at"`
	DecidedBy   string   `json:"decided_by,omitempty"`
	DecidedAt   string   `json:"decided_at,omitempty"`
	Decision    Decision `json:"decision,omitempty"`
	Note        string   `json:"note,omitempty"`
}
