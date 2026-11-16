package mwanachamaforms_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

func TestReleasingAMemberUnlinksTheirLinkAnswersOnly(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	o, ok := s.Object("respondent")
	if !ok {
		t.Fatal("no respondent object")
	}
	table := s.TableFor(o)
	insert := fmt.Sprintf("insert into %s (id, public_key, first_seen_at, claimed_by_member_id, claimed_at) values (?, ?, '2026-01-01T00:00:00Z', ?, '2026-01-02T00:00:00Z')", table)
	for _, row := range [][]string{{"r-1", "k-1", "mem-1"}, {"r-2", "k-2", "mem-2"}} {
		if err := db.Exec(insert, row[0], row[1], row[2]).Error; err != nil {
			t.Fatalf("seed respondent: %v", err)
		}
	}
	if err := m.ReleaseMember(ctx, "mem-1"); err != nil {
		t.Fatalf("ReleaseMember: %v", err)
	}
	claimed := func(id string) string {
		var who *string
		if err := db.Raw("select claimed_by_member_id from "+table+" where id = ?", id).Scan(&who).Error; err != nil {
			t.Fatalf("read respondent: %v", err)
		}
		if who == nil {
			return ""
		}
		return *who
	}
	if got := claimed("r-1"); got != "" {
		t.Fatalf("released respondent still claimed by %q", got)
	}
	if got := claimed("r-2"); got != "mem-2" {
		t.Fatalf("other member's respondent claimed by %q, want mem-2", got)
	}
}

func TestReleasingNobodyIsRefused(t *testing.T) {
	m := gatedManager(t, nil)
	if err := m.ReleaseMember(context.Background(), ""); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("ReleaseMember empty = %v, want ErrInvalidAnswer", err)
	}
}
