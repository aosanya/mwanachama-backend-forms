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

type PublicFormView struct {
	ID                  string         `json:"id"`
	Title               string         `json:"title"`
	Status              Status         `json:"status"`
	OpensAt             string         `json:"opens_at,omitempty"`
	ClosesAt            string         `json:"closes_at"`
	Language            string         `json:"language,omitempty"`
	WindowAdvisory      bool           `json:"window_advisory"`
	Standing            bool           `json:"standing"`
	OneVoice            OneVoice       `json:"one_voice,omitempty"`
	PublishedAt         string         `json:"published_at,omitempty"`
	PublisherName       string         `json:"publisher_name,omitempty"`
	PublishingAuthority string         `json:"publishing_authority,omitempty"`
	Audience            Audience       `json:"audience"`
	CollectionMode      CollectionMode `json:"collection_mode"`
	Schedule            FormSchedule   `json:"schedule,omitzero"`
	CreatedAt           string         `json:"created_at"`
	LastUpdated         string         `json:"last_updated"`
}

type PublicLinkView struct {
	ID        string     `json:"id"`
	FormID    string     `json:"form_id"`
	Key       string     `json:"key"`
	Label     string     `json:"label,omitempty"`
	Status    LinkStatus `json:"status"`
	CreatedAt string     `json:"created_at"`
}

type ResolvedLinkView struct {
	Link PublicLinkView `json:"link"`
	Form PublicFormView `json:"form"`
}

func (r ResolvedLink) RespondentView() ResolvedLinkView {
	f, l := r.Form, r.Link
	return ResolvedLinkView{
		Link: PublicLinkView{ID: l.ID, FormID: l.FormID, Key: l.Key, Label: l.Label, Status: l.Status, CreatedAt: l.CreatedAt},
		Form: PublicFormView{
			ID: f.ID, Title: f.Title, Status: f.Status, OpensAt: f.OpensAt, ClosesAt: f.ClosesAt, Language: f.Language,
			WindowAdvisory: f.WindowAdvisory, Standing: f.Standing, OneVoice: f.OneVoice, PublishedAt: f.PublishedAt,
			PublisherName: f.PublisherName, PublishingAuthority: f.PublishingAuthority, Audience: f.Audience,
			CollectionMode: f.CollectionMode, Schedule: f.Schedule, CreatedAt: f.CreatedAt, LastUpdated: f.LastUpdated,
		},
	}
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

type Registration struct {
	ID           string `json:"id"`
	RespondentID string `json:"respondent_id"`
	FormID       string `json:"form_id"`
	RegisteredAt string `json:"registered_at"`
}

type Claim struct {
	ID           string `json:"id"`
	RespondentID string `json:"respondent_id"`
	FormID       string `json:"form_id"`
	ClaimedAt    string `json:"claimed_at"`
}

type DeclarationCount struct {
	ChapterID string `json:"chapter_id"`
	Count     int    `json:"count"`
}
