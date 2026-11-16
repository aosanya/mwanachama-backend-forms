package models

import (
	"strings"
	"testing"
)

func num(v float64) *float64 { return &v }
func whole(v int) *int       { return &v }

func options(ids ...string) []QuestionOption {
	out := make([]QuestionOption, len(ids))
	for i, id := range ids {
		out[i] = QuestionOption{ID: id, QuestionID: "q1", Ordinal: i + 1, Label: id}
	}
	return out
}

func TestAnswerIsRefusedWithTheReasonAPersonCanFix(t *testing.T) {
	yes := true
	cases := []struct {
		name     string
		question Question
		options  []QuestionOption
		answer   Answer
		refusal  string
	}{
		{"yes_no with no value", Question{AnswerType: AnswerYesNo}, nil, Answer{}, "requires value_bool"},
		{"yes_no with a value", Question{AnswerType: AnswerYesNo}, nil, Answer{ValueBool: &yes}, ""},
		{"single choice with two picks", Question{AnswerType: AnswerSingleChoice}, options("a", "b"), Answer{OptionIDs: []string{"a", "b"}}, "exactly one option_id"},
		{"single choice with none", Question{AnswerType: AnswerSingleChoice}, options("a"), Answer{}, "exactly one option_id"},
		{"single choice with a stranger", Question{AnswerType: AnswerSingleChoice}, options("a"), Answer{OptionIDs: []string{"z"}}, "does not belong to this question"},
		{"single choice picked", Question{AnswerType: AnswerSingleChoice}, options("a", "b"), Answer{OptionIDs: []string{"b"}}, ""},
		{"multi select with none", Question{AnswerType: AnswerMultiSelect}, options("a"), Answer{}, "at least one option_id"},
		{"multi select picking one twice", Question{AnswerType: AnswerMultiSelect}, options("a", "b"), Answer{OptionIDs: []string{"a", "a"}}, "picked more than once"},
		{"multi select below the minimum", Question{AnswerType: AnswerMultiSelect, MinSelections: whole(2)}, options("a", "b", "c"), Answer{OptionIDs: []string{"a"}}, "fewer than min_selections 2"},
		{"multi select above the maximum", Question{AnswerType: AnswerMultiSelect, MaxSelections: whole(1)}, options("a", "b"), Answer{OptionIDs: []string{"a", "b"}}, "more than max_selections 1"},
		{"multi select inside the bounds", Question{AnswerType: AnswerMultiSelect, MinSelections: whole(1), MaxSelections: whole(2)}, options("a", "b", "c"), Answer{OptionIDs: []string{"a", "c"}}, ""},
		{"free text of spaces", Question{AnswerType: AnswerFreeText}, nil, Answer{ValueText: "   "}, "requires value_text"},
		{"free text too long counts letters not bytes", Question{AnswerType: AnswerFreeText, MaxLength: whole(3)}, nil, Answer{ValueText: "ñandú"}, "exceeds max_length 3"},
		{"free text exactly at the length", Question{AnswerType: AnswerFreeText, MaxLength: whole(3)}, nil, Answer{ValueText: "ñan"}, ""},
		{"date that is not a date", Question{AnswerType: AnswerDate}, nil, Answer{ValueDate: "10/11/2026"}, "YYYY-MM-DD"},
		{"date before the minimum", Question{AnswerType: AnswerDate, MinDate: "2026-11-10"}, nil, Answer{ValueDate: "2026-11-09"}, "before min_date 2026-11-10"},
		{"date after the maximum", Question{AnswerType: AnswerDate, MaxDate: "2026-11-10"}, nil, Answer{ValueDate: "2026-11-11"}, "after max_date 2026-11-10"},
		{"date on the maximum", Question{AnswerType: AnswerDate, MaxDate: "2026-11-10"}, nil, Answer{ValueDate: "2026-11-10"}, ""},
		{"time that is not a time", Question{AnswerType: AnswerTime}, nil, Answer{ValueTime: "9am"}, "HH:MM"},
		{"time", Question{AnswerType: AnswerTime}, nil, Answer{ValueTime: "09:30"}, ""},
		{"number with no value", Question{AnswerType: AnswerNumber}, nil, Answer{}, "requires value_number"},
		{"currency below the minimum", Question{AnswerType: AnswerCurrency, MinValue: num(10)}, nil, Answer{ValueNumber: num(9.5)}, "below min_value 10"},
		{"number above the maximum", Question{AnswerType: AnswerNumber, MaxValue: num(100)}, nil, Answer{ValueNumber: num(100.5)}, "above max_value 100"},
		{"number on the minimum", Question{AnswerType: AnswerNumber, MinValue: num(10)}, nil, Answer{ValueNumber: num(10)}, ""},
		{"an unknown type", Question{AnswerType: "scribble"}, nil, Answer{}, `unknown answer_type "scribble"`},
		{"files on a text question", Question{AnswerType: AnswerFreeText}, nil, Answer{ValueText: "x", FileIDs: []string{"f1"}}, "file_ids do not fit a free_text question"},
	}
	for _, c := range cases {
		err := ValidateAnswer(c.question, c.options, c.answer)
		if c.refusal == "" {
			if err != nil {
				t.Errorf("%s: refused: %v", c.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.refusal) {
			t.Errorf("%s: want a refusal containing %q, got %v", c.name, c.refusal, err)
		}
	}
}

func TestPhotoAndFileAnswersNeedFilesWithinTheLimit(t *testing.T) {
	photo := Question{AnswerType: AnswerPhoto}
	file := Question{AnswerType: AnswerFile}
	cases := []struct {
		name     string
		question Question
		answer   Answer
		def      int
		refusal  string
	}{
		{"no file", photo, Answer{}, 0, "requires at least one file_id"},
		{"an empty id", file, Answer{FileIDs: []string{"f1", ""}}, 0, "file_id must not be empty"},
		{"the same file twice", file, Answer{FileIDs: []string{"f1", "f1"}}, 0, `"f1" given more than once`},
		{"over the default limit", file, Answer{FileIDs: []string{"f1", "f2", "f3"}}, 2, "more than the limit of 2"},
		{"over the question's own limit beats the default", Question{AnswerType: AnswerFile, MaxFiles: whole(1)}, Answer{FileIDs: []string{"f1", "f2"}}, 5, "more than the limit of 1"},
		{"the question's own limit lifts the default", Question{AnswerType: AnswerFile, MaxFiles: whole(3)}, Answer{FileIDs: []string{"f1", "f2", "f3"}}, 1, ""},
		{"no limit at all", file, Answer{FileIDs: []string{"f1", "f2", "f3", "f4"}}, 0, ""},
		{"a photo that must be taken now but was uploaded", Question{AnswerType: AnswerPhoto, RequireCapture: true}, Answer{FileIDs: []string{"f1", "f2"}, CapturedFileIDs: []string{"f1"}}, 0, `"f2" was not taken now`},
		{"a photo taken now", Question{AnswerType: AnswerPhoto, RequireCapture: true}, Answer{FileIDs: []string{"f1"}, CapturedFileIDs: []string{"f1"}}, 0, ""},
	}
	for _, c := range cases {
		err := ValidateAnswerWithin(c.question, nil, c.answer, c.def)
		if c.refusal == "" {
			if err != nil {
				t.Errorf("%s: refused: %v", c.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.refusal) {
			t.Errorf("%s: want a refusal containing %q, got %v", c.name, c.refusal, err)
		}
	}
}

func TestFileLimitPrefersTheQuestionsOwn(t *testing.T) {
	if got := FileLimit(Question{}, 4); got != 4 {
		t.Errorf("default: %d", got)
	}
	if got := FileLimit(Question{MaxFiles: whole(2)}, 4); got != 2 {
		t.Errorf("own: %d", got)
	}
	if got := FileLimit(Question{}, 0); got != 0 {
		t.Errorf("none: %d", got)
	}
}

func TestAQuestionIsRefusedWhenItsRulesDoNotFitItsType(t *testing.T) {
	cases := []struct {
		name     string
		question Question
		refusal  string
	}{
		{"number bounds on text", Question{AnswerType: AnswerFreeText, MinValue: num(1)}, "min_value and max_value do not apply to a free_text question"},
		{"date bounds on a number", Question{AnswerType: AnswerNumber, MaxDate: "2026-01-01"}, "min_date and max_date do not apply to a number question"},
		{"selection bounds on single choice", Question{AnswerType: AnswerSingleChoice, MinSelections: whole(1)}, "min_selections and max_selections do not apply"},
		{"max_files on a number", Question{AnswerType: AnswerNumber, MaxFiles: whole(2)}, "max_files does not apply to a number question"},
		{"capture on a file", Question{AnswerType: AnswerFile, RequireCapture: true}, "require_capture applies only to a photo question"},
		{"max_files of zero", Question{AnswerType: AnswerFile, MaxFiles: whole(0)}, "max_files must be at least 1"},
		{"min above max value", Question{AnswerType: AnswerCurrency, MinValue: num(5), MaxValue: num(4)}, "min_value 5 is above max_value 4"},
		{"a date bound that is not a date", Question{AnswerType: AnswerDate, MinDate: "soon"}, "min_date must be YYYY-MM-DD"},
		{"min date after max date", Question{AnswerType: AnswerDate, MinDate: "2026-02-01", MaxDate: "2026-01-01"}, "min_date 2026-02-01 is after max_date 2026-01-01"},
		{"selection minimum of zero", Question{AnswerType: AnswerMultiSelect, MinSelections: whole(0)}, "min_selections must be at least 1"},
		{"selection maximum of zero", Question{AnswerType: AnswerMultiSelect, MaxSelections: whole(0)}, "max_selections must be at least 1"},
		{"selection min above max", Question{AnswerType: AnswerMultiSelect, MinSelections: whole(3), MaxSelections: whole(2)}, "min_selections 3 is above max_selections 2"},
	}
	for _, c := range cases {
		err := ValidateValidators(c.question)
		if err == nil || !strings.Contains(err.Error(), c.refusal) {
			t.Errorf("%s: want a refusal containing %q, got %v", c.name, c.refusal, err)
		}
	}
	fine := []Question{
		{AnswerType: AnswerNumber, MinValue: num(1), MaxValue: num(1)},
		{AnswerType: AnswerDate, MinDate: "2026-01-01", MaxDate: "2026-01-01"},
		{AnswerType: AnswerMultiSelect, MinSelections: whole(1), MaxSelections: whole(1)},
		{AnswerType: AnswerPhoto, RequireCapture: true, MaxFiles: whole(2)},
		{AnswerType: AnswerFreeText},
	}
	for i, q := range fine {
		if err := ValidateValidators(q); err != nil {
			t.Errorf("fine[%d]: %v", i, err)
		}
	}
}

func TestUnansweredRequiredLeavesOutAnsweredAndSupersededQuestions(t *testing.T) {
	questions := []Question{
		{ID: "q1", Required: true},
		{ID: "q2", Required: true},
		{ID: "q3", Required: false},
		{ID: "q4", Required: true, SupersedesQuestionID: "q2"},
		{ID: "q5", Required: true},
	}
	answers := []Answer{{QuestionID: "q1"}, {QuestionID: "q3"}}
	got := UnansweredRequired(questions, answers)
	var ids []string
	for _, q := range got {
		ids = append(ids, q.ID)
	}
	if strings.Join(ids, ",") != "q4,q5" {
		t.Errorf("unanswered: %v", ids)
	}
	if left := UnansweredRequired(nil, nil); len(left) != 0 {
		t.Errorf("nothing to answer: %v", left)
	}
}
