package mwanachamaforms

import "testing"

func TestScratchBlueprintParses(t *testing.T) {
	s, err := SpecFor("form")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	for _, o := range s.Objects {
		t.Logf("%s -> %s", o.Role, s.TableFor(o))
	}
	for _, stmt := range s.DDL("postgres") {
		t.Log(stmt)
	}
}
