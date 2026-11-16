package models

type ResultReport struct {
	ID          string   `json:"id"`
	FormID      string   `json:"form_id"`
	ChapterID   string   `json:"chapter_id"`
	ChapterIDs  []string `json:"chapter_ids"`
	Version     int      `json:"version"`
	GitPath     string   `json:"git_path"`
	GitCommit   string   `json:"git_commit"`
	PublishedBy string   `json:"published_by"`
	PublishedAt string   `json:"published_at"`
}

type ResultReportView struct {
	ID          string   `json:"id"`
	FormID      string   `json:"form_id"`
	ChapterID   string   `json:"chapter_id"`
	ChapterIDs  []string `json:"chapter_ids"`
	Version     int      `json:"version"`
	GitPath     string   `json:"git_path"`
	GitCommit   string   `json:"git_commit"`
	PublishedAt string   `json:"published_at"`
}

type ResultReportRead interface{ resultReportRead() }

func (ResultReport) resultReportRead() {}

func (ResultReportView) resultReportRead() {}

func (r ResultReport) ReadAs(narrowed bool) ResultReportRead {
	if narrowed {
		return r.RespondentView()
	}
	return r
}

func (r ResultReport) RespondentView() ResultReportView {
	return ResultReportView{
		ID: r.ID, FormID: r.FormID, ChapterID: r.ChapterID, ChapterIDs: r.ChapterIDs, Version: r.Version,
		GitPath: r.GitPath, GitCommit: r.GitCommit, PublishedAt: r.PublishedAt,
	}
}
