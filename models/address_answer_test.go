package models

import (
	"strings"
	"testing"
)

func TestAnAddressAnswerIsAPickedPlaceOrFreeTextWhereTheQuestionAllowsIt(t *testing.T) {
	strict := Question{AnswerType: AnswerAddress}
	loose := Question{AnswerType: AnswerAddress, AllowFreeText: true}
	cases := []struct {
		name     string
		question Question
		answer   Answer
		refusal  string
	}{{"a picked address", strict, Answer{ValueText: "Fő tér 2", PlaceRef: "address:a1"}, ""}, {"a picked street", strict, Answer{ValueText: "Fő tér", PlaceRef: "street:Fő tér"}, ""}, {"a picked address on a question that allows free text", loose, Answer{ValueText: "Fő tér 2", PlaceRef: "address:a1"}, ""}, {"free text where it is allowed", loose, Answer{ValueText: "the car park behind the mill"}, ""}, {"free text where it is not allowed", strict, Answer{ValueText: "the car park behind the mill"}, "free text"}, {"no words at all", loose, Answer{PlaceRef: "address:a1"}, "requires value_text"}, {"words of spaces", loose, Answer{ValueText: "   "}, "requires value_text"}, {"a reference of no known kind", strict, Answer{ValueText: "Fő tér 2", PlaceRef: "house:a1"}, "place_ref"}, {"an address reference with nothing after it", strict, Answer{ValueText: "Fő tér 2", PlaceRef: "address:"}, "place_ref"}, {"a street reference with nothing after it", strict, Answer{ValueText: "Fő tér", PlaceRef: "street:"}, "place_ref"}}
	for _, c := range cases {
		err := ValidateAnswer(c.question, nil, c.answer)
		if c.refusal == "" {
			if err != nil {
				t.Errorf("%s: refused: %v", c.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.refusal) {
			t.Errorf("%s: want a refusal naming %q, got %v", c.name, c.refusal, err)
		}
	}
}
func TestAPlaceRefOnAnAnswerOfAnotherTypeIsRefused(t *testing.T) {
	err := ValidateAnswer(Question{AnswerType: AnswerFreeText}, nil, Answer{ValueText: "Fő tér 2", PlaceRef: "address:a1"})
	if err == nil || !strings.Contains(err.Error(), "place_ref") {
		t.Fatalf("want a refusal naming place_ref, got %v", err)
	}
}
func TestAllowFreeTextBelongsToAnAddressQuestionOnly(t *testing.T) {
	if err := ValidateValidators(Question{AnswerType: AnswerAddress, AllowFreeText: true}); err != nil {
		t.Fatalf("an address question may allow free text: %v", err)
	}
	err := ValidateValidators(Question{AnswerType: AnswerFreeText, AllowFreeText: true})
	if err == nil || !strings.Contains(err.Error(), "allow_free_text") {
		t.Fatalf("want a refusal naming allow_free_text, got %v", err)
	}
}
func TestAnAddressAnswerLandsAsTheWordsNotTheReference(t *testing.T) {
	source := LandingSource{Question: Question{AnswerType: AnswerAddress}, Answer: Answer{ValueText: "Fő tér 2", PlaceRef: "address:a1"}}
	if got := source.value(); got != "Fő tér 2" {
		t.Fatalf("an address lands as %v, want the words", got)
	}
}
