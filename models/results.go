package models

type ResultReport struct {
	ID          string `json:"id"`
	FormID      string `json:"form_id"`
	ChapterID   string `json:"chapter_id"`
	Version     int    `json:"version"`
	GitPath     string `json:"git_path"`
	GitCommit   string `json:"git_commit"`
	PublishedBy string `json:"published_by"`
	PublishedAt string `json:"published_at"`
}
