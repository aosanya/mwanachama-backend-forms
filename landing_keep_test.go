package mwanachamaforms_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type checkingWriter struct {
	collectingWriter
	allowed map[string]bool
}

func (w *checkingWriter) CheckFields(_ string, fields []string) error {
	for _, f := range fields {
		if !w.allowed[f] {
			return errors.New("no such field " + f)
		}
	}
	return nil
}

func TestAnEditThatLeavesOutLandsInKeepsIt(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	r := openableRound(t, m, personLandsIn())
	edit := models.Form{ID: r.form.ID, Title: "Renamed", ClosesAt: r.form.ClosesAt, Audience: r.form.Audience, CollectionMode: r.form.CollectionMode}
	updated, err := m.Update(ctx, edit)
	if err != nil {
		t.Fatalf("Update = %v", err)
	}
	if updated.Title != "Renamed" || updated.LandsIn.Record != "catalog.person" {
		t.Fatalf("after edit title=%q lands_in=%+v, want it kept", updated.Title, updated.LandsIn)
	}
	got, _ := m.Get(ctx, r.form.ID)
	if got.LandsIn.Record != "catalog.person" || got.LandsIn.FieldMap["Ward"] != "ward" {
		t.Fatalf("stored lands_in = %+v, want it kept", got.LandsIn)
	}
}

func TestARoundMappingAFieldTheModuleDoesNotHaveCannotBeOpened(t *testing.T) {
	m := gatedManager(t, nil)
	w := &checkingWriter{allowed: map[string]bool{"full_name": true, "ward": true}}
	m.UseRecordWriters(writersFor("catalog", w))
	landsIn := personLandsIn()
	landsIn.FieldMap["Has a phone"] = "has_phnoe"
	r := openableRound(t, m, landsIn, "has a phone")
	_, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", "")
	if !errors.Is(err, forms.ErrInvalidForm) || !strings.Contains(err.Error(), "has_phnoe") {
		t.Fatalf("Publish = %v, want ErrInvalidForm naming has_phnoe", err)
	}
	got, _ := m.Get(context.Background(), r.form.ID)
	if got.Status != models.StatusDraft {
		t.Fatalf("status = %s, want draft", got.Status)
	}
}

func TestARoundMappingOnlyFieldsTheModuleHasOpens(t *testing.T) {
	m := gatedManager(t, nil)
	w := &checkingWriter{allowed: map[string]bool{"full_name": true, "ward": true, "has_phone": true}}
	m.UseRecordWriters(writersFor("catalog", w))
	r := openableRound(t, m, personLandsIn(), "has a phone")
	if _, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish = %v", err)
	}
}

func TestATestRunChecksTheFieldsToo(t *testing.T) {
	m := gatedManager(t, nil)
	w := &checkingWriter{allowed: map[string]bool{"full_name": true}}
	m.UseRecordWriters(writersFor("catalog", w))
	run, _ := m.ForRun("run-1")
	r := openableRound(t, run, personLandsIn(), "has a phone")
	if _, _, err := run.Publish(context.Background(), r.form.ID, "", "author-1", ""); !errors.Is(err, forms.ErrInvalidForm) {
		t.Fatalf("Publish in a run = %v, want ErrInvalidForm", err)
	}
}
