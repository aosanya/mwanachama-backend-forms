package models

import (
	"fmt"
	"time"
)

const DateLayout = "2006-01-02"

func ValidateValidators(q Question) error {
	if err := boundsOnlyFor(q, q.MinValue != nil || q.MaxValue != nil, "min_value and max_value", AnswerNumber, AnswerCurrency); err != nil {
		return err
	}
	if err := boundsOnlyFor(q, q.MinDate != "" || q.MaxDate != "", "min_date and max_date", AnswerDate); err != nil {
		return err
	}
	if err := boundsOnlyFor(q, q.MinSelections != nil || q.MaxSelections != nil, "min_selections and max_selections", AnswerMultiSelect); err != nil {
		return err
	}
	if err := validateFileRules(q); err != nil {
		return err
	}
	if q.MinValue != nil && q.MaxValue != nil && *q.MinValue > *q.MaxValue {
		return fmt.Errorf("min_value %v is above max_value %v", *q.MinValue, *q.MaxValue)
	}
	if err := validateDateBounds(q.MinDate, q.MaxDate); err != nil {
		return err
	}
	return validateSelectionBounds(q.MinSelections, q.MaxSelections)
}

func boundsOnlyFor(q Question, set bool, names string, types ...AnswerType) error {
	if !set {
		return nil
	}
	for _, t := range types {
		if q.AnswerType == t {
			return nil
		}
	}
	return fmt.Errorf("%s do not apply to a %s question", names, q.AnswerType)
}

func validateDateBounds(minDate, maxDate string) error {
	for name, v := range map[string]string{"min_date": minDate, "max_date": maxDate} {
		if v == "" {
			continue
		}
		if _, err := time.Parse(DateLayout, v); err != nil {
			return fmt.Errorf("%s must be YYYY-MM-DD: %w", name, err)
		}
	}
	if minDate != "" && maxDate != "" && minDate > maxDate {
		return fmt.Errorf("min_date %s is after max_date %s", minDate, maxDate)
	}
	return nil
}

func validateSelectionBounds(minSel, maxSel *int) error {
	if minSel != nil && *minSel < 1 {
		return fmt.Errorf("min_selections must be at least 1")
	}
	if maxSel != nil && *maxSel < 1 {
		return fmt.Errorf("max_selections must be at least 1")
	}
	if minSel != nil && maxSel != nil && *minSel > *maxSel {
		return fmt.Errorf("min_selections %d is above max_selections %d", *minSel, *maxSel)
	}
	return nil
}

func checkValueBounds(q Question, v float64) error {
	if q.MinValue != nil && v < *q.MinValue {
		return fmt.Errorf("value_number %v is below min_value %v", v, *q.MinValue)
	}
	if q.MaxValue != nil && v > *q.MaxValue {
		return fmt.Errorf("value_number %v is above max_value %v", v, *q.MaxValue)
	}
	return nil
}

func checkDateBounds(q Question, d string) error {
	if q.MinDate != "" && d < q.MinDate {
		return fmt.Errorf("value_date %s is before min_date %s", d, q.MinDate)
	}
	if q.MaxDate != "" && d > q.MaxDate {
		return fmt.Errorf("value_date %s is after max_date %s", d, q.MaxDate)
	}
	return nil
}

func checkSelectionBounds(q Question, picked int) error {
	if q.MinSelections != nil && picked < *q.MinSelections {
		return fmt.Errorf("%d option_ids picked, fewer than min_selections %d", picked, *q.MinSelections)
	}
	if q.MaxSelections != nil && picked > *q.MaxSelections {
		return fmt.Errorf("%d option_ids picked, more than max_selections %d", picked, *q.MaxSelections)
	}
	return nil
}

func UnansweredRequired(questions []Question, answers []Answer) []Question {
	superseded := map[string]bool{}
	for _, q := range questions {
		if q.SupersedesQuestionID != "" {
			superseded[q.SupersedesQuestionID] = true
		}
	}
	answered := map[string]bool{}
	for _, a := range answers {
		answered[a.QuestionID] = true
	}
	var out []Question
	for _, q := range questions {
		if q.Required && !superseded[q.ID] && !answered[q.ID] {
			out = append(out, q)
		}
	}
	return out
}
