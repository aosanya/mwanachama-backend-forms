package models

// Target names one chapter a Form is aimed at (member-audience forms only —
// a public-audience form carries no Target rows, see [ErrPublicFormNoTargets]).
// IncludesDescendants controls whether the target reaches the named
// chapter's sub-chapters too, or that chapter alone — see
// UserManager.ListReachable's doc for exactly how the flag is read.
//
// CreatedAt does not exist on the gateway's survey.Target — added here for
// the same reason Form gained one: this repo's UUID ids carry no
// chronological order, so ListTargets needs a real timestamp to sort by
// instead of the gateway's "ORDER BY id" (which tracked insertion order only
// because the gateway's ids are sequential).
type Target struct {
	ID                  string `json:"id"`
	FormID              string `json:"form_id"`
	ChapterID           string `json:"chapter_id"`
	IncludesDescendants bool   `json:"includes_descendants"`
	CreatedAt           string `json:"created_at"`
}
