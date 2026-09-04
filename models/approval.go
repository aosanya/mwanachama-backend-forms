package models

// Decision is how an Approval was decided.
type Decision string

const (
	DecisionApproved Decision = "approved"
	DecisionRefused  Decision = "refused"
)

// Approval is one submit-for-approval cycle on a Form. RequestedAt/DecidedAt
// are [TimeLayout] strings; DecidedAt empty means the approval is still
// open (awaiting a decision) — see form_impl.go's decide/openApproval doc.
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
