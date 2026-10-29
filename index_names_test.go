package mwanachamaforms_test

import (
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const realLengthInstanceID = "agy0d6e4079"

func TestEveryFormsTableAndIndexNameFitsPostgresWithItsTestCopy(t *testing.T) {
	s := testSpec(t, realLengthInstanceID)
	for _, o := range s.Objects {
		for _, name := range []string{s.TableFor(o), s.TestTableFor(o)} {
			if len(name) > spec.MaxIdentifier {
				t.Errorf("table %q is %d bytes, over the %d Postgres truncates at", name, len(name), spec.MaxIdentifier)
			}
		}
		for _, idx := range o.Indexes {
			for _, name := range []string{s.IndexFor(o, idx), s.TestIndexFor(o, idx)} {
				if len(name) > spec.MaxIdentifier {
					t.Errorf("index %q is %d bytes, over the %d Postgres truncates at", name, len(name), spec.MaxIdentifier)
				}
			}
		}
	}
}

func TestTheFormsSpecLoadsUnderARealLengthInstanceID(t *testing.T) {
	if len(realLengthInstanceID) != len("agy")+8 {
		t.Fatalf("instance id %q is not the length wakala-api mints", realLengthInstanceID)
	}
	if _, err := mwanachamaforms.SpecFor(realLengthInstanceID); err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
}
