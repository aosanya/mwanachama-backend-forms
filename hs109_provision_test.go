package mwanachamaforms_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestProvisionAddsLockedAfterSendToAFormTableThatPredatesIt(t *testing.T) {
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
	ctx := context.Background()
	closes := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	earlier, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Census", ClosesAt: closes})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	o, _ := s.ByRole("form")
	table := s.TableFor(o)
	if err := db.Exec("alter table " + table + " drop column locked_after_send").Error; err != nil {
		t.Fatalf("drop locked_after_send: %v", err)
	}

	hasColumn := func() bool {
		var found []int
		if err := db.Raw("select 1 from pragma_table_info(?) where name = ?", table, "locked_after_send").Scan(&found).Error; err != nil {
			t.Fatalf("read columns: %v", err)
		}
		return len(found) > 0
	}
	if hasColumn() {
		t.Fatal("old shape still has locked_after_send")
	}

	for run := 1; run <= 2; run++ {
		if err := mwanachamaforms.Provision(db, s); err != nil {
			t.Fatalf("Provision run %d: %v", run, err)
		}
		if !hasColumn() {
			t.Fatalf("run %d: locked_after_send missing", run)
		}
	}

	got, err := mgr.Get(ctx, earlier.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.LockedAfterSend {
		t.Fatalf("earlier form reads locked: %+v", got)
	}
	locked, err := mgr.Create(ctx, mwanachamaforms.Form{Title: "Once", ClosesAt: closes, LockedAfterSend: true})
	if err != nil {
		t.Fatalf("Create locked: %v", err)
	}
	if again, err := mgr.Get(ctx, locked.ID); err != nil || !again.LockedAfterSend {
		t.Fatalf("Get locked = %+v, %v", again, err)
	}
}
