package models

// AnswerType is the shape of value a Question expects back.
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

// HasOptions is true for the two answer shapes that carry QuestionOption rows.
func (t AnswerType) HasOptions() bool {
	return t == AnswerSingleChoice || t == AnswerMultiSelect
}

// Question is one prompt on a Form. Mirrors the gateway's
// internal/domain/survey.Question field for field, with SurveyID renamed to
// FormID.
//
// Version/SupersedesQuestionID implement in-place-for-drafts,
// version-chain-for-published editing: UpdateQuestion (draft only) mutates a
// Question row directly; VersionQuestion (published only) leaves the
// predecessor row untouched and creates a new row pointing back at it via
// SupersedesQuestionID — see form_impl.go's VersionQuestion doc.
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

// QuestionOption is one choice a single_choice/multi_select Question offers.
type QuestionOption struct {
	ID         string `json:"id"`
	QuestionID string `json:"question_id"`
	Ordinal    int    `json:"ordinal"`
	Label      string `json:"label"`
}

// AddOptionResult is AddOption's return shape: the new option plus the
// before/after option counts read from the same locked write. Unlike the
// gateway's survey.AddOptionResult, this carries no CustodyEventID/
// CustodyOccurredAt — this repo does not depend on the gateway's
// custody/act-log domain, the same considered exclusion
// mwanachama-backend-actor already made for Deregister (see this repo's
// CLAUDE.md).
type AddOptionResult struct {
	Option        QuestionOption `json:"option"`
	OptionsBefore int            `json:"options_before"`
	OptionsAfter  int            `json:"options_after"`
}

// VersionQuestionResult is VersionQuestion's return shape: the new Question
// row plus the options written against it. Same custody omission as
// [AddOptionResult].
type VersionQuestionResult struct {
	Question Question         `json:"question"`
	Options  []QuestionOption `json:"options"`
}
