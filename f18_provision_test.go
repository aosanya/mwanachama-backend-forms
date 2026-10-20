package mwanachamaforms_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestProvisionAddsThePublisherColumnsToAFormTableThatPredatesThem(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s := testSpec(t, "test")
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	o, _ := s.ByRole("form")
	table := s.TableFor(o)
	for _, column := range []string{"publisher_name", "publishing_authority"} {
		if err := db.Exec("alter table " + table + " drop column " + column).Error; err != nil {
			t.Fatalf("drop %s: %v", column, err)
		}
	}

	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision over the older table: %v", err)
	}
	mgr, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	mgr.UseNames(func(context.Context, string) (string, error) { return "Wanjiku Kamau", nil })
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{
		Title: "Census", ClosesAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := mgr.Publish(ctx, f.ID, "", "clerk-1", "Ward office"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	got, err := mgr.Get(ctx, f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.PublisherName != "Wanjiku Kamau" || got.PublishingAuthority != "Ward office" {
		t.Fatalf("publisher_name %q, publishing_authority %q", got.PublisherName, got.PublishingAuthority)
	}
}
