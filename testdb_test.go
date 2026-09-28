package mwanachamaforms_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

// testSpec is the shipped domain spec under a test instance, so these tests
// run against the same declaration the real mount does rather than a
// table set that exists only here.
func testSpec(t *testing.T, instance string) *spec.Spec {
	t.Helper()
	s, err := mwanachamaforms.SpecFor(instance)
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	return s
}

func newTestManager(t *testing.T) mwanachamaforms.FormManager {
	t.Helper()
	mgr, _ := newTestManagerDB(t)
	return mgr
}

// newTestManagerDB also hands back the database, for the tests that have to
// plant a row the manager would refuse to write.
func newTestManagerDB(t *testing.T) (mwanachamaforms.FormManager, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	s := testSpec(t, "test")
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	return mgr, db
}

func TestNewFormManager_NilDB(t *testing.T) {
	if _, err := mwanachamaforms.NewFormManager(nil, testSpec(t, "test")); err == nil {
		t.Fatal("expected error for nil db")
	}
}
