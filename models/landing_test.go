package models

import (
	"encoding/json"
	"testing"
)

func TestOnlyAStandingFormMayNameALanding(t *testing.T) {
	cases := []struct {
		standing bool
		landing  Landing
		refused  bool
	}{
		{true, LandingThread, false},
		{true, LandingCase, false},
		{true, LandingNone, false},
		{false, LandingNone, false},
		{false, LandingThread, true},
		{false, LandingCase, true},
	}
	for _, c := range cases {
		err := ValidateLanding(c.standing, c.landing)
		if (err != nil) != c.refused {
			t.Fatalf("standing=%v landing=%q: err = %v, refused should be %v", c.standing, c.landing, err, c.refused)
		}
	}
}

func TestAFormLeavesLandingOutOfItsJSONWhenItHasNone(t *testing.T) {
	var none, named map[string]any
	raw, _ := json.Marshal(Form{Standing: true})
	if err := json.Unmarshal(raw, &none); err != nil {
		t.Fatal(err)
	}
	if _, present := none["landing"]; present {
		t.Fatalf("a form landing nowhere carries landing: %s", raw)
	}
	raw, _ = json.Marshal(Form{Standing: true, Landing: LandingCase})
	if err := json.Unmarshal(raw, &named); err != nil {
		t.Fatal(err)
	}
	if named["landing"] != "case" {
		t.Fatalf("landing = %v in %s, want case", named["landing"], raw)
	}
}
