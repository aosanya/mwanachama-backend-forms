package mwanachamaforms_test

import (
	"context"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestASecondSendBySameCallerLeavesOneSubmissionAndTheStoreRefusesADouble(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Once"})
	for range 3 {
		if err := submitOne(mgr, f.ID, q); err != nil {
			t.Fatalf("SubmitAnswers: %v", err)
		}
	}
	got, err := mgr.ListSubmissions(ctx, f.ID, "", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(got) != 1 || got[0].MemberID != "alice" || got[0].Answers != 1 || got[0].EditedAt == "" {
		t.Fatalf("sends = %+v, want alice's one, edited", got)
	}

	s := testSpec(t, "test")
	o, _ := s.ByRole("submission")
	table := s.TableFor(o)
	cols := "form_id, member_id, chapter_id, period_key, submitted_at, edited_at, collector, collected_at, channel, language"
	dup := db.Exec("insert into " + table + " (id, " + cols + ") select 'double', " + cols + " from " + table)
	if dup.Error == nil || !strings.Contains(strings.ToLower(dup.Error.Error()), "unique") {
		t.Fatalf("a second submission row for the same form, member, group and period: %v, want a unique refusal", dup.Error)
	}
}
