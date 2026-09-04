package mwanachamaforms_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

// newTestManager builds a [mwanachamaforms.FormManager] backed by a fresh
// in-memory sqlite database, migrated via [mwanachamaforms.Migrate] — the
// same pattern mwanachama-backend-actor's testdb_test.go uses.
func newTestManager(t *testing.T) mwanachamaforms.FormManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	tables := mwanachamaforms.DefaultTableNames("test")
	if err := mwanachamaforms.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	mgr, err := mwanachamaforms.NewFormManager(db, tables)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	return mgr
}

func TestNewFormManager_NilDB(t *testing.T) {
	if _, err := mwanachamaforms.NewFormManager(nil, mwanachamaforms.DefaultTableNames("test")); err == nil {
		t.Fatal("expected error for nil db")
	}
}
