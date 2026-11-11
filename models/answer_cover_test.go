package models

import (
	"strings"
	"testing"
)

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }
func ptrB(v bool) *bool       { return &v }

var abcOptions = []QuestionOption{{ID: "a"}, {ID: "b"}, {ID: "c"}}

func TestValidateAnswerAcceptsAndRefusesByAnswerType(t *testing.T) {
	cases := []struct {
		name    string
		q       Question
		a       Answer
		refusal string
	}{
		{"yes_no with a value", Question{AnswerType: AnswerYesNo}, Answer{ValueBool: ptrB(false)}, ""},
		{"yes_no without a value", Question{AnswerType: AnswerYesNo}, Answer{}, "requires value_bool"},
		{"single_choice with one option", Question{AnswerType: AnswerSingleChoice}, Answer{OptionIDs: []string{"a"}}, ""},
		{"single_choice with two options", Question{AnswerType: AnswerSingleChoice}, Answer{OptionIDs: []string{"a", "b"}}, "exactly one option_id"},
		{"single_choice with none", Question{AnswerType: AnswerSingleChoice}, Answer{}, "exactly one option_id"},
		{"single_choice with a foreign option", Question{AnswerType: AnswerSingleChoice}, Answer{OptionIDs: []string{"z"}}, "does not belong"},
		{"multi_select none", Question{AnswerType: AnswerMultiSelect}, Answer{}, "at least one option_id"},
		{"multi_select duplicate", Question{AnswerType: AnswerMultiSelect}, Answer{OptionIDs: []string{"a", "a"}}, "more than once"},
		{"multi_select below minimum", Question{AnswerType: AnswerMultiSelect, MinSelections: ptrI(2)}, Answer{OptionIDs: []string{"a"}}, "fewer than min_selections 2"},
		{"multi_select above maximum", Question{AnswerType: AnswerMultiSelect, MaxSelections: ptrI(1)}, Answer{OptionIDs: []string{"a", "b"}}, "more than max_selections 1"},
		{"multi_select within bounds", Question{AnswerType: AnswerMultiSelect, MinSelections: ptrI(1), MaxSelections: ptrI(2)}, Answer{OptionIDs: []string{"a", "c"}}, ""},
		{"free_text blank", Question{AnswerType: AnswerFreeText}, Answer{ValueText: "  \t"}, "requires value_text"},
		{"free_text at the limit counts runes", Question{AnswerType: AnswerFreeText, MaxLength: ptrI(3)}, Answer{ValueText: "éèê"}, ""},
		{"free_text over the limit", Question{AnswerType: AnswerFreeText, MaxLength: ptrI(3)}, Answer{ValueText: "éèêa"}, "exceeds max_length 3"},
		{"date malformed", Question{AnswerType: AnswerDate}, Answer{ValueDate: "12/05/2026"}, "YYYY-MM-DD"},
		{"date before minimum", Question{AnswerType: AnswerDate, MinDate: "2026-01-01"}, Answer{ValueDate: "2025-12-31"}, "before min_date"},
		{"date after maximum", Question{AnswerType: AnswerDate, MaxDate: "2026-01-01"}, Answer{ValueDate: "2026-01-02"}, "after max_date"},
		{"date on the boundary", Question{AnswerType: AnswerDate, MinDate: "2026-01-01", MaxDate: "2026-01-01"}, Answer{ValueDate: "2026-01-01"}, ""},
		{"time malformed", Question{AnswerType: AnswerTime}, Answer{ValueTime: "25:00"}, "HH:MM"},
		{"time valid", Question{AnswerType: AnswerTime}, Answer{ValueTime: "23:59"}, ""},
		{"number missing", Question{AnswerType: AnswerNumber}, Answer{}, "number answer requires value_number"},
		{"currency missing", Question{AnswerType: AnswerCurrency}, Answer{}, "currency answer requires value_number"},
		{"number below minimum", Question{AnswerType: AnswerNumber, MinValue: ptrF(5)}, Answer{ValueNumber: ptrF(4.9)}, "below min_value"},
		{"number above maximum", Question{AnswerType: AnswerCurrency, MaxValue: ptrF(5)}, Answer{ValueNumber: ptrF(5.1)}, "above max_value"},
		{"number zero is a value", Question{AnswerType: AnswerNumber, MinValue: ptrF(0)}, Answer{ValueNumber: ptrF(0)}, ""},
		{"unknown type", Question{AnswerType: "hologram"}, Answer{}, `unknown answer_type "hologram"`},
		{"files on a text question", Question{AnswerType: AnswerFreeText}, Answer{ValueText: "x", FileIDs: []string{"f"}}, "do not fit a free_text question"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateAnswer(c.q, abcOptions, c.a)
			if c.refusal == "" {
				if err != nil {
					t.Fatalf("want accepted, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.refusal) {
				t.Fatalf("want refusal containing %q, got %v", c.refusal, err)
			}
		})
	}
}

func TestFileAnswersAreHeldToCountDuplicatesAndCapture(t *testing.T) {
	photo := Question{AnswerType: AnswerPhoto}
	cases := []struct {
		name       string
		q          Question
		a          Answer
		defaultMax int
		refusal    string
	}{
		{"no files", photo, Answer{}, 0, "requires at least one file_id"},
		{"blank file id", photo, Answer{FileIDs: []string{""}}, 0, "must not be empty"},
		{"repeated file id", photo, Answer{FileIDs: []string{"f", "f"}}, 0, "given more than once"},
		{"over the default limit", photo, Answer{FileIDs: []string{"a", "b"}}, 1, "more than the limit of 1"},
		{"question limit beats the default", Question{AnswerType: AnswerFile, MaxFiles: ptrI(2)}, Answer{FileIDs: []string{"a", "b"}}, 1, ""},
		{"no limit when none set", photo, Answer{FileIDs: []string{"a", "b", "c"}}, 0, ""},
		{"capture required but not captured", Question{AnswerType: AnswerPhoto, RequireCapture: true}, Answer{FileIDs: []string{"a", "b"}, CapturedFileIDs: []string{"a"}}, 0, `"b" was not taken now`},
		{"capture required and captured", Question{AnswerType: AnswerPhoto, RequireCapture: true}, Answer{FileIDs: []string{"a"}, CapturedFileIDs: []string{"a"}}, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateAnswerWithin(c.q, nil, c.a, c.defaultMax)
			if c.refusal == "" && err != nil {
				t.Fatalf("want accepted, got %v", err)
			}
			if c.refusal != "" && (err == nil || !strings.Contains(err.Error(), c.refusal)) {
				t.Fatalf("want refusal containing %q, got %v", c.refusal, err)
			}
		})
	}
}

func TestFileLimitPrefersTheQuestionOverTheDefault(t *testing.T) {
	if got := FileLimit(Question{MaxFiles: ptrI(4)}, 9); got != 4 {
		t.Fatalf("got %d, want 4", got)
	}
	if got := FileLimit(Question{}, 9); got != 9 {
		t.Fatalf("got %d, want 9", got)
	}
}

func TestValidateValidatorsRefusesBoundsThatDoNotFitTheQuestion(t *testing.T) {
	cases := []struct {
		name    string
		q       Question
		refusal string
	}{
		{"no bounds", Question{AnswerType: AnswerFreeText}, ""},
		{"value bounds on text", Question{AnswerType: AnswerFreeText, MinValue: ptrF(1)}, "min_value and max_value do not apply to a free_text"},
		{"date bounds on number", Question{AnswerType: AnswerNumber, MaxDate: "2026-01-01"}, "min_date and max_date do not apply to a number"},
		{"selection bounds on single choice", Question{AnswerType: AnswerSingleChoice, MaxSelections: ptrI(2)}, "min_selections and max_selections do not apply"},
		{"min above max value", Question{AnswerType: AnswerNumber, MinValue: ptrF(3), MaxValue: ptrF(2)}, "min_value 3 is above max_value 2"},
		{"equal value bounds", Question{AnswerType: AnswerCurrency, MinValue: ptrF(2), MaxValue: ptrF(2)}, ""},
		{"bad date bound", Question{AnswerType: AnswerDate, MinDate: "soon"}, "min_date must be YYYY-MM-DD"},
		{"dates reversed", Question{AnswerType: AnswerDate, MinDate: "2026-02-01", MaxDate: "2026-01-01"}, "is after max_date"},
		{"min selections zero", Question{AnswerType: AnswerMultiSelect, MinSelections: ptrI(0)}, "min_selections must be at least 1"},
		{"max selections zero", Question{AnswerType: AnswerMultiSelect, MaxSelections: ptrI(0)}, "max_selections must be at least 1"},
		{"selections reversed", Question{AnswerType: AnswerMultiSelect, MinSelections: ptrI(3), MaxSelections: ptrI(2)}, "min_selections 3 is above max_selections 2"},
		{"max files on text", Question{AnswerType: AnswerFreeText, MaxFiles: ptrI(1)}, "max_files does not apply"},
		{"capture on a file question", Question{AnswerType: AnswerFile, RequireCapture: true}, "only to a photo question"},
		{"zero max files", Question{AnswerType: AnswerPhoto, MaxFiles: ptrI(0)}, "max_files must be at least 1"},
		{"photo with capture and limit", Question{AnswerType: AnswerPhoto, RequireCapture: true, MaxFiles: ptrI(3)}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateValidators(c.q)
			if c.refusal == "" && err != nil {
				t.Fatalf("want accepted, got %v", err)
			}
			if c.refusal != "" && (err == nil || !strings.Contains(err.Error(), c.refusal)) {
				t.Fatalf("want refusal containing %q, got %v", c.refusal, err)
			}
		})
	}
}

func TestUnansweredRequiredSkipsSupersededAndAnsweredQuestions(t *testing.T) {
	questions := []Question{
		{ID: "old", Required: true},
		{ID: "new", Required: true, SupersedesQuestionID: "old"},
		{ID: "optional"},
		{ID: "done", Required: true},
		{ID: "missing", Required: true},
	}
	answers := []Answer{{QuestionID: "done"}}
	var got []string
	for _, q := range UnansweredRequired(questions, answers) {
		got = append(got, q.ID)
	}
	if strings.Join(got, ",") != "new,missing" {
		t.Fatalf("got %v, want [new missing]", got)
	}
}

func TestQuestionTypeHelpers(t *testing.T) {
	for _, tp := range []AnswerType{AnswerPhoto, AnswerFile} {
		if !tp.HoldsFiles() || tp.HasOptions() {
			t.Fatalf("%s should hold files and no options", tp)
		}
	}
	for _, tp := range []AnswerType{AnswerSingleChoice, AnswerMultiSelect} {
		if tp.HoldsFiles() || !tp.HasOptions() {
			t.Fatalf("%s should hold options and no files", tp)
		}
	}
	if AnswerFreeText.HoldsFiles() || AnswerFreeText.HasOptions() {
		t.Fatal("free_text holds neither")
	}
}
