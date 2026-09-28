package models

type LinkStatus string

const (
	LinkActive  LinkStatus = "active"
	LinkRetired LinkStatus = "retired"
)

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

type ResolvedLink struct {
	Link PublicLink `json:"link"`
	Form Form       `json:"form"`
}

type Respondent struct {
	ID                string `json:"id"`
	PublicKey         string `json:"public_key,omitempty"`
	FirstSeenAt       string `json:"first_seen_at"`
	ClaimedByMemberID string `json:"claimed_by_member_id,omitempty"`
	ClaimedAt         string `json:"claimed_at,omitempty"`
}

type RespondentPublicView struct {
	ID          string `json:"id"`
	PublicKey   string `json:"public_key,omitempty"`
	FirstSeenAt string `json:"first_seen_at"`
}

func (r Respondent) PublicView() RespondentPublicView {
	return RespondentPublicView{ID: r.ID, PublicKey: r.PublicKey, FirstSeenAt: r.FirstSeenAt}
}

type Declaration struct {
	ID                string `json:"id"`
	RespondentID      string `json:"respondent_id"`
	FormID            string `json:"form_id"`
	DeclaredChapterID string `json:"declared_chapter_id,omitempty"`
	DeclaredText      string `json:"declared_text,omitempty"`
	CreatedAt         string `json:"created_at"`
}
