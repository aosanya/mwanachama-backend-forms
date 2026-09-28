package models

import (
	"fmt"
	"strings"
	"time"
)

type Answer struct {
	ID         string   `json:"id"`
	FormID     string   `json:"form_id"`
	QuestionID string   `json:"question_id"`
	MemberID   string   `json:"member_id"`
	ChapterID  string   `json:"chapter_id,omitempty"`
	OptionIDs  []string `json:"option_ids,omitempty"`

	ValueText   string   `json:"value_text,omitempty"`
	ValueNumber *float64 `json:"value_number,omitempty"`
	ValueDate   string   `json:"value_date,omitempty"`
	ValueTime   string   `json:"value_time,omitempty"`
	ValueBool   *bool    `json:"value_bool,omitempty"`

	AnsweredAt string `json:"answered_at"`
	EditedAt   string `json:"edited_at,omitempty"`
}

func ValidateAnswer(q Question, options []QuestionOption, a Answer) error {
	switch q.AnswerType {
	case AnswerYesNo:
		if a.ValueBool == nil {
			return fmt.Errorf("yes_no answer requires value_bool")
		}
	case AnswerSingleChoice:
		if len(a.OptionIDs) != 1 {
			return fmt.Errorf("single_choice answer requires exactly one option_id")
		}
		return validateOptions(options, a.OptionIDs)
	case AnswerMultiSelect:
		if len(a.OptionIDs) == 0 {
			return fmt.Errorf("multi_select answer requires at least one option_id")
		}
		return validateOptions(options, a.OptionIDs)
	case AnswerFreeText:
		if strings.TrimSpace(a.ValueText) == "" {
			return fmt.Errorf("free_text answer requires value_text")
		}
		if q.MaxLength != nil && len([]rune(a.ValueText)) > *q.MaxLength {
			return fmt.Errorf("value_text exceeds max_length %d", *q.MaxLength)
		}
	case AnswerDate:
		if _, err := time.Parse("2006-01-02", a.ValueDate); err != nil {
			return fmt.Errorf("date answer requires value_date as YYYY-MM-DD: %w", err)
		}
	case AnswerTime:
		if _, err := time.Parse("15:04", a.ValueTime); err != nil {
			return fmt.Errorf("time answer requires value_time as HH:MM: %w", err)
		}
	case AnswerNumber, AnswerCurrency:
		if a.ValueNumber == nil {
			return fmt.Errorf("%s answer requires value_number", q.AnswerType)
		}
	default:
		return fmt.Errorf("unknown answer_type %q", q.AnswerType)
	}
	return nil
}

func validateOptions(options []QuestionOption, picked []string) error {
	seen := map[string]bool{}
	for _, id := range picked {
		if seen[id] {
			return fmt.Errorf("option_id %q picked more than once", id)
		}
		seen[id] = true
		found := false
		for _, o := range options {
			if o.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("option_id %q does not belong to this question", id)
		}
	}
	return nil
}
