package models

type AnswerType string

const (
	AnswerYesNo        AnswerType = "yes_no"
	AnswerSingleChoice AnswerType = "single_choice"
	AnswerMultiSelect  AnswerType = "multi_select"
	AnswerFreeText     AnswerType = "free_text"
	AnswerDate         AnswerType = "date"
	AnswerTime         AnswerType = "time"
	AnswerNumber       AnswerType = "number"
	AnswerCurrency     AnswerType = "currency"
)

func (t AnswerType) HasOptions() bool {
	return t == AnswerSingleChoice || t == AnswerMultiSelect
}

type Question struct {
	ID          string     `json:"id"`
	FormID      string     `json:"form_id"`
	Ordinal     int        `json:"ordinal"`
	AnswerType  AnswerType `json:"answer_type"`
	Prompt      string     `json:"prompt"`
	Placeholder string     `json:"placeholder,omitempty"`
	MaxLength   *int       `json:"max_length,omitempty"`
	UnitLabel   string     `json:"unit_label,omitempty"`
	Helper      string     `json:"helper,omitempty"`

	CurrencyCode string    `json:"currency_code,omitempty"`
	QuickPicks   []float64 `json:"quick_picks,omitempty"`
	YesLabel     string    `json:"yes_label,omitempty"`
	NoLabel      string    `json:"no_label,omitempty"`

	Version              int    `json:"version"`
	SupersedesQuestionID string `json:"supersedes_question_id,omitempty"`
}

type QuestionOption struct {
	ID         string `json:"id"`
	QuestionID string `json:"question_id"`
	Ordinal    int    `json:"ordinal"`
	Label      string `json:"label"`
}

type QuestionDraft struct {
	Question
	Options []QuestionOption `json:"options,omitempty"`
}

type AddOptionResult struct {
	Option        QuestionOption `json:"option"`
	OptionsBefore int            `json:"options_before"`
	OptionsAfter  int            `json:"options_after"`
}

type VersionQuestionResult struct {
	Question Question         `json:"question"`
	Options  []QuestionOption `json:"options"`
}
