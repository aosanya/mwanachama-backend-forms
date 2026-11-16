package mwanachamaforms_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/consent"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type gateFor map[string]error

func (g gateFor) Check(_ context.Context, actorID string) error { return g[actorID] }

func gatedManager(t *testing.T, g consent.Gate) forms.FormManager {
	t.Helper()
	m, _, _ := gatedManagerOn(t, g)
	return m
}

func gatedManagerOn(t *testing.T, g consent.Gate) (forms.FormManager, *gorm.DB, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := forms.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := forms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	m, err := forms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	m.UseConsentGate(g)
	return m, db, s
}

func TestAnAnswerNeedsALiveConsent(t *testing.T) {
	m := gatedManager(t, gateFor{"mem-2": consent.ErrRequired, "mem-3": consent.ErrWithdrawn})
	ctx := context.Background()
	if _, err := m.SubmitAnswers(ctx, "f-1", "mem-2", nil); !errors.Is(err, forms.ErrConsentRequired) {
		t.Fatalf("answer without consent = %v, want ErrConsentRequired", err)
	}
	if _, err := m.SubmitAnswersFor(ctx, "f-1", "mem-3", "", "", "", nil); !errors.Is(err, forms.ErrConsentWithdrawn) {
		t.Fatalf("answer after withdrawal = %v, want ErrConsentWithdrawn", err)
	}
	if _, err := m.SubmitAnswers(ctx, "f-1", "mem-1", nil); errors.Is(err, forms.ErrConsentRequired) || errors.Is(err, forms.ErrConsentWithdrawn) {
		t.Fatalf("consenting answer refused for consent: %v", err)
	}
}

func TestADeclarationFromAClaimedRespondentNeedsALiveConsent(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, gateFor{"mem-2": consent.ErrWithdrawn})
	o, _ := s.Object("respondent")
	insert := fmt.Sprintf("insert into %s (id, public_key, first_seen_at, claimed_by_member_id, claimed_at) values (?, ?, '2026-01-01T00:00:00Z', ?, '2026-01-02T00:00:00Z')", s.TableFor(o))
	for _, row := range [][]string{{"r-1", "k-1", "mem-1"}, {"r-2", "k-2", "mem-2"}, {"r-3", "k-3", ""}} {
		if err := db.Exec(insert, row[0], row[1], row[2]).Error; err != nil {
			t.Fatalf("seed respondent: %v", err)
		}
	}
	if _, err := m.Declare(ctx, models.Declaration{RespondentID: "r-2", FormID: "f-1", DeclaredChapterID: "g-1"}); !errors.Is(err, forms.ErrConsentWithdrawn) {
		t.Fatalf("declaration after withdrawal = %v, want ErrConsentWithdrawn", err)
	}
	for _, id := range []string{"r-1", "r-3"} {
		if _, err := m.Declare(ctx, models.Declaration{RespondentID: id, FormID: "f-1", DeclaredChapterID: "g-1"}); err != nil {
			t.Fatalf("declaration by %s = %v", id, err)
		}
	}
}
