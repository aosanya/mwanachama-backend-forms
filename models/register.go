package models

type RegisterQuery struct {
	Search    string `query:"q"`
	ChapterID string `query:"chapter_id"`
	Status    Status `query:"status"`
	Limit     *int   `query:"limit"`
	Offset    int    `query:"offset"`
}

type RegisterRow struct {
	Form        Form `json:"form"`
	Questions   int  `json:"questions"`
	Respondents int  `json:"respondents"`
}

type RegisterPage struct {
	Rows        []RegisterRow  `json:"rows"`
	Total       int            `json:"total"`
	ByStatus    map[Status]int `json:"by_status"`
	Respondents int            `json:"respondents"`
}
