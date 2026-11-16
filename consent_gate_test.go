package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/consent"
)

type gateFor map[string]error

func (g gateFor) Check(_ context.Context, actorID string) error { return g[actorID] }

func gatedManager(t *testing.T, g consent.Gate) forms.FormManager {
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
	return m
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
