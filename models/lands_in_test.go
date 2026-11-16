package models

import (
	"strings"
	"testing"
)

func consultationLandsIn() LandsIn {
	return LandsIn{
		Record:  "response",
		KeyedBy: []string{"consultation_reference", "ward", "person_or_anonymous"},
		FieldMap: map[string]string{
			"Ward":              "ward",
			"Do you agree?":     "agrees",
			"In your own words": "free_text",
			"Which street":      "street",
		},
	}
}

func consultationSources() []LandingSource {
	yes := true
	return []LandingSource{
		{Question: Question{Prompt: "Ward", AnswerType: AnswerSingleChoice},
			Options: []QuestionOption{{ID: "o1", Label: "Kertváros"}, {ID: "o2", Label: "Óváros"}},
			Answer:  Answer{OptionIDs: []string{"o1"}}},
		{Question: Question{Prompt: "Do you agree?", AnswerType: AnswerYesNo}, Answer: Answer{ValueBool: &yes}},
		{Question: Question{Prompt: "in your own words", AnswerType: AnswerFreeText}, Answer: Answer{ValueText: "More lighting on Malom utca"}},
	}
}

func TestLandsInComposesARecordKeyedByContextAndMappedFields(t *testing.T) {
	landed, err := consultationLandsIn().Land(consultationSources(), map[string]string{
		"consultation_reference": "C-2026-04",
		"person_or_anonymous":    "anonymous",
	})
	if err != nil {
		t.Fatal(err)
	}
	if landed.Record != "response" {
		t.Fatalf("record = %q", landed.Record)
	}
	want := map[string]string{"consultation_reference": "C-2026-04", "ward": "Kertváros", "person_or_anonymous": "anonymous"}
	for part, v := range want {
		if landed.Key[part] != v {
			t.Fatalf("key %s = %q, want %q", part, landed.Key[part], v)
		}
	}
	if landed.Fields["agrees"] != true || landed.Fields["free_text"] != "More lighting on Malom utca" {
		t.Fatalf("fields = %v", landed.Fields)
	}
}

func TestLandsInRefusesAQuestionWithNoField(t *testing.T) {
	sources := append(consultationSources(), LandingSource{Question: Question{Prompt: "Your phone number", AnswerType: AnswerFreeText}})
	_, err := consultationLandsIn().Land(sources, map[string]string{"consultation_reference": "C-1", "person_or_anonymous": "anonymous"})
	if err == nil || !strings.Contains(err.Error(), "Your phone number") {
		t.Fatalf("err = %v", err)
	}
}

func TestLandsInRefusesAKeyPartWithNoValue(t *testing.T) {
	_, err := consultationLandsIn().Land(consultationSources(), map[string]string{"consultation_reference": "C-1"})
	if err == nil || !strings.Contains(err.Error(), "person_or_anonymous") {
		t.Fatalf("err = %v", err)
	}
}

func TestLandsInValidateNeedsRecordKeyAndFields(t *testing.T) {
	for name, l := range map[string]LandsIn{
		"no record": {KeyedBy: []string{"a"}, FieldMap: map[string]string{"q": "f"}},
		"no key":    {Record: "r", FieldMap: map[string]string{"q": "f"}},
		"no fields": {Record: "r", KeyedBy: []string{"a"}},
		"blank":     {Record: "r", KeyedBy: []string{"a"}, FieldMap: map[string]string{"q": " "}},
	} {
		if l.Validate() == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestLandsInListsUnmappedQuestionsSorted(t *testing.T) {
	got := consultationLandsIn().UnmappedQuestions([]Question{{Prompt: "Zebra"}, {Prompt: "Ward"}, {Prompt: "Alpha"}})
	if strings.Join(got, ",") != "Alpha,Zebra" {
		t.Fatalf("got %v", got)
	}
}
