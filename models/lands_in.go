package models

import (
	"fmt"
	"sort"
	"strings"
)

type LandsIn struct {
	Record   string            `json:"record"`
	KeyedBy  []string          `json:"keyed_by"`
	FieldMap map[string]string `json:"field_map"`
}

type LandedRecord struct {
	Record string            `json:"record"`
	Key    map[string]string `json:"key"`
	Fields map[string]any    `json:"fields"`
}

type LandingSource struct {
	Question Question
	Options  []QuestionOption
	Answer   Answer
}

func (l LandsIn) Validate() error {
	if strings.TrimSpace(l.Record) == "" {
		return fmt.Errorf("lands_in needs a record")
	}
	if len(l.KeyedBy) == 0 {
		return fmt.Errorf("lands_in needs at least one key part")
	}
	if len(l.FieldMap) == 0 {
		return fmt.Errorf("lands_in needs a field_map")
	}
	for prompt, field := range l.FieldMap {
		if strings.TrimSpace(prompt) == "" || strings.TrimSpace(field) == "" {
			return fmt.Errorf("lands_in field_map entries need a question and a field")
		}
	}
	return nil
}

func (l LandsIn) fieldFor(prompt string) (string, bool) {
	want := strings.ToLower(strings.TrimSpace(prompt))
	for question, field := range l.FieldMap {
		if strings.ToLower(strings.TrimSpace(question)) == want {
			return field, true
		}
	}
	return "", false
}

func (l LandsIn) UnmappedQuestions(questions []Question) []string {
	var out []string
	for _, q := range questions {
		if _, ok := l.fieldFor(q.Prompt); !ok {
			out = append(out, q.Prompt)
		}
	}
	sort.Strings(out)
	return out
}

func (s LandingSource) value() any {
	a := s.Answer
	switch s.Question.AnswerType {
	case AnswerYesNo:
		if a.ValueBool != nil {
			return *a.ValueBool
		}
	case AnswerNumber, AnswerCurrency:
		if a.ValueNumber != nil {
			return *a.ValueNumber
		}
	case AnswerDate:
		return a.ValueDate
	case AnswerTime:
		return a.ValueTime
	case AnswerSingleChoice, AnswerMultiSelect:
		labels := make([]string, 0, len(a.OptionIDs))
		for _, id := range a.OptionIDs {
			labels = append(labels, optionLabel(s.Options, id))
		}
		if s.Question.AnswerType == AnswerSingleChoice && len(labels) == 1 {
			return labels[0]
		}
		return labels
	case AnswerPhoto, AnswerFile:
		return a.FileIDs
	}
	return a.ValueText
}

func optionLabel(options []QuestionOption, id string) string {
	for _, o := range options {
		if o.ID == id {
			return o.Label
		}
	}
	return id
}

func (l LandsIn) Land(sources []LandingSource, context map[string]string) (LandedRecord, error) {
	if err := l.Validate(); err != nil {
		return LandedRecord{}, err
	}
	questions := make([]Question, 0, len(sources))
	for _, s := range sources {
		questions = append(questions, s.Question)
	}
	if missing := l.UnmappedQuestions(questions); len(missing) > 0 {
		return LandedRecord{}, fmt.Errorf("questions with no field in lands_in: %s", strings.Join(missing, ", "))
	}
	fields := make(map[string]any, len(sources))
	for _, s := range sources {
		field, _ := l.fieldFor(s.Question.Prompt)
		fields[field] = s.value()
	}
	key := make(map[string]string, len(l.KeyedBy))
	for _, part := range l.KeyedBy {
		if v, ok := context[part]; ok && v != "" {
			key[part] = v
			continue
		}
		if v, ok := fields[part]; ok && fmt.Sprint(v) != "" {
			key[part] = fmt.Sprint(v)
			continue
		}
		return LandedRecord{}, fmt.Errorf("lands_in key part %q has no value", part)
	}
	return LandedRecord{Record: l.Record, Key: key, Fields: fields}, nil
}
