package models

// RegisterQuery filters and pages [RegisterPage]. Search matches a
// case-insensitive substring of Form.Title only. Limit nil returns every
// matching row; a non-nil Limit of zero returns no rows but still computes
// Total/ByStatus/Respondents over the full filtered set.
type RegisterQuery struct {
	Search    string
	ChapterID string
	Status    Status
	Limit     *int
	Offset    int
}

// RegisterRow is one Form in a [RegisterPage], with its live question and
// respondent counts attached.
type RegisterRow struct {
	Form        Form `json:"form"`
	Questions   int  `json:"questions"`
	Respondents int  `json:"respondents"`
}

// RegisterPage is [UserManager.Register]'s page: Total/ByStatus/Respondents
// are computed over the whole filtered (pre-pagination) set, so
// Total == the sum of ByStatus always, even when Rows is a shorter page.
type RegisterPage struct {
	Rows        []RegisterRow  `json:"rows"`
	Total       int            `json:"total"`
	ByStatus    map[Status]int `json:"by_status"`
	Respondents int            `json:"respondents"`
}
