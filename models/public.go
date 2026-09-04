package models

// LinkStatus is whether a PublicLink still resolves.
type LinkStatus string

const (
	LinkActive  LinkStatus = "active"
	LinkRetired LinkStatus = "retired"
)

// PublicLink is the shareable, no-account entry point for a public-audience
// Form, created once on Publish. Key is the caller-supplied short link
// token — this package stores it as given and does not generate or
// deduplicate it itself (mirrors the gateway's memory store exactly; see
// form_impl.go's Publish doc).
type PublicLink struct {
	ID        string     `json:"id"`
	FormID    string     `json:"form_id"`
	Key       string     `json:"key"`
	Label     string     `json:"label,omitempty"`
	Status    LinkStatus `json:"status"`
	CreatedAt string     `json:"created_at"`
	RetiredAt string     `json:"retired_at,omitempty"`
	CreatedBy string     `json:"created_by,omitempty"`
}

// Respondent is an anonymous public-link answerer, identified by PublicKey
// (a client-held token, not a member id). ClaimedByMemberID/ClaimedAt exist
// for schema parity with the gateway's survey.Respondent, but — exactly as
// upstream — no method in this package ever sets them; the claim-on-signup
// flow they describe is not implemented here either.
type Respondent struct {
	ID                string `json:"id"`
	PublicKey         string `json:"public_key,omitempty"`
	FirstSeenAt       string `json:"first_seen_at"`
	ClaimedByMemberID string `json:"claimed_by_member_id,omitempty"`
	ClaimedAt         string `json:"claimed_at,omitempty"`
}

// RespondentPublicView is Respondent with ClaimedByMemberID/ClaimedAt
// dropped — never expose which member (if any) a respondent turned out to
// be on a public route.
type RespondentPublicView struct {
	ID          string `json:"id"`
	PublicKey   string `json:"public_key,omitempty"`
	FirstSeenAt string `json:"first_seen_at"`
}

// PublicView strips the fields a public route must never return.
func (r Respondent) PublicView() RespondentPublicView {
	return RespondentPublicView{ID: r.ID, PublicKey: r.PublicKey, FirstSeenAt: r.FirstSeenAt}
}

// Declaration is a respondent's self-reported chapter for one Form, unique
// per (RespondentID, FormID) — a repeat Declare overwrites in place.
type Declaration struct {
	ID                string `json:"id"`
	RespondentID      string `json:"respondent_id"`
	FormID            string `json:"form_id"`
	DeclaredChapterID string `json:"declared_chapter_id,omitempty"`
	DeclaredText      string `json:"declared_text,omitempty"`
	CreatedAt         string `json:"created_at"`
}
