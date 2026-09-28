package models

type Target struct {
	ID                  string `json:"id"`
	FormID              string `json:"form_id"`
	ChapterID           string `json:"chapter_id"`
	IncludesDescendants bool   `json:"includes_descendants"`
	CreatedAt           string `json:"created_at"`
}
