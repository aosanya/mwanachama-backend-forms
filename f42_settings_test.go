package mwanachamaforms_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func provisionedSettingsDB(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s := testSpec(t, "test")
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	return db, s
}

func TestTheBlueprintDeclaresFormsSettings(t *testing.T) {
	s := testSpec(t, "test")
	approval, ok := s.Setting(mwanachamaforms.SettingApprovalRequired)
	if !ok || approval.Type != spec.TypeBool || approval.Default != false {
		t.Fatalf("approval_required = %+v, %v", approval, ok)
	}
	types, ok := s.Setting(mwanachamaforms.SettingFormTypes)
	if !ok || types.Type != spec.TypeJSON {
		t.Fatalf("form_types = %+v, %v", types, ok)
	}
}

func TestTheDefaultsKeepTodaysBehaviour(t *testing.T) {
	db, s := provisionedSettingsDB(t)
	approval, err := spec.ResolveStored(db, s, mwanachamaforms.SettingApprovalRequired)
	if err != nil || approval.Value != false || approval.Layer != spec.LayerModule {
		t.Fatalf("approval_required = %+v, %v", approval, err)
	}
	mgr, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	if got := mgr.FormTypes(); len(got) != 0 {
		t.Fatalf("FormTypes = %+v, want none", got)
	}
}

func TestADefinitionAndAnOverrideResolveThroughTheManager(t *testing.T) {
	db, s := provisionedSettingsDB(t)
	err := spec.PutDefinitionSettings(db, s, map[string]any{
		mwanachamaforms.SettingFormTypes: []any{
			map[string]any{"name": "Survey", "audience": "member", "collection_mode": "self"},
			map[string]any{"name": "Census", "audience": "public", "collection_mode": "interviewer"},
		},
	})
	if err != nil {
		t.Fatalf("PutDefinitionSettings: %v", err)
	}
	if _, err := spec.StoreOverride(db, s, mwanachamaforms.SettingApprovalRequired, true, "owner-1", "2026-10-24T08:00:00Z"); err != nil {
		t.Fatalf("StoreOverride: %v", err)
	}
	approval, err := spec.ResolveStored(db, s, mwanachamaforms.SettingApprovalRequired)
	if err != nil || approval.Value != true || approval.Layer != spec.LayerOverride || approval.By != "owner-1" {
		t.Fatalf("approval_required = %+v, %v", approval, err)
	}
	mgr, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	want := []models.FormType{
		{Name: "Survey", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf},
		{Name: "Census", Audience: models.AudiencePublic, CollectionMode: models.CollectionInterviewer},
	}
	got := mgr.FormTypes()
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("FormTypes = %+v, want %+v", got, want)
	}
}

func TestAFormTypeThatBreaksTheAudienceRuleIsRefused(t *testing.T) {
	db, s := provisionedSettingsDB(t)
	err := spec.PutDefinitionSettings(db, s, map[string]any{
		mwanachamaforms.SettingFormTypes: []any{
			map[string]any{"name": "Interview", "audience": "member", "collection_mode": "interviewer"},
		},
	})
	if err != nil {
		t.Fatalf("PutDefinitionSettings: %v", err)
	}
	if _, err := mwanachamaforms.NewFormManager(db, s); err == nil {
		t.Fatal("NewFormManager accepted a member-audience interviewer type")
	}
}
