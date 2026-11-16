package mwanachamaforms_test

import (
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestADatabaseHoldingTwoVoicesFromBeforeTheKeyStillStarts(t *testing.T) {
	r, race := voiceRaceRound(t, models.OneVoiceRefuse)
	table := race.tableOf(t, "submission")
	if err := race.db.Exec("drop index " + table + "_one_voice_slot_idx").Error; err != nil {
		t.Fatalf("drop the voice index: %v", err)
	}
	if err := race.db.Exec("alter table " + table + " drop column voice_key").Error; err != nil {
		t.Fatalf("drop the voice key: %v", err)
	}
	for i, group := range []string{"g-1", "g-2"} {
		row := map[string]any{
			"id": "old-" + group, "form_id": r.form.ID, "member_id": "mem-1", "chapter_id": group, "period_key": "",
			"repeat_of": "", "withdrawn_at": "", "submitted_at": "2026-01-01T00:00:00Z", "edited_at": "", "collector": "mem-1",
			"collected_at": "", "channel": "app", "language": "", "paid_at": "", "paid_by": "",
		}
		if err := race.db.Table(table).Create(row).Error; err != nil {
			t.Fatalf("keep voice %d from before the key: %v", i, err)
		}
	}
	if err := forms.Provision(race.db, race.s); err != nil {
		t.Fatalf("Provision over two voices from before the key = %v, want it to start", err)
	}
	if err := r.sendAs("mem-2", "g-1", "keep the market open"); err != nil {
		t.Fatalf("a send after the upgrade: %v", err)
	}
	if err := r.sendAs("mem-2", "g-2", "close the market"); err == nil {
		t.Fatal("a second voice after the upgrade was kept, want it refused")
	}
}
