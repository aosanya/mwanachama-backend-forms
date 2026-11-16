package mwanachamaforms_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const clashSlug = "agency1"

func clashDB(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := forms.SpecFor(clashSlug)
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	return db, s
}

func mustProvision(t *testing.T, db *gorm.DB, s *spec.Spec) {
	t.Helper()
	if err := forms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
}

func rowsIn(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
