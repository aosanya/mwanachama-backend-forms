package mwanachamaforms_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/postgres"
)

const (
	twoProcessRounds = 6
	twoProcessCopies = 16
)

func sqliteFileDB(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path+"?_pragma=journal_mode(WAL)"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func postgresDBOrSkip(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test")
	}
	sqlDB, err := postgres.Open(context.Background(), postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("postgres.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

func sendTogetherAcross(r voiceRound, managers []forms.FormManager, copies int) []error {
	failures := make([]error, copies)
	start := make(chan struct{})
	var wg sync.WaitGroup
	groups := []string{"g-1", "g-2"}
	for i := range failures {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, failures[i] = managers[i%len(managers)].SubmitAnswersAs(context.Background(), r.form.ID, "mem-1", groups[(i/len(managers))%len(groups)],
				[]models.Answer{{QuestionID: r.question.ID, ValueText: "view"}})
		}()
	}
	close(start)
	wg.Wait()
	return failures
}

func voicesHeld(t *testing.T, m forms.FormManager, formID string) int {
	t.Helper()
	listed, err := m.ListSubmissions(context.Background(), formID, "mem-1", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	held := 0
	for _, s := range listed {
		if s.RepeatOf == "" {
			held++
		}
	}
	return held
}

func twoProcessVoiceRace(t *testing.T, first, second *gorm.DB, instance string, voice models.OneVoice) {
	t.Helper()
	s := testSpec(t, instance)
	if err := forms.Provision(first, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			_ = first.Migrator().DropTable(s.TableFor(o))
		}
	})
	a, err := forms.NewFormManager(first, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	b, err := forms.NewFormManager(second, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	for round := 0; round < twoProcessRounds; round++ {
		r := openRoundForTwoGroups(t, a, voice)
		kept := 0
		failures := sendTogetherAcross(r, []forms.FormManager{a, b}, twoProcessCopies)
		for _, err := range failures {
			switch {
			case err == nil:
				kept++
			case errors.Is(err, forms.ErrSecondVoice) && voice == models.OneVoiceRefuse:
			default:
				t.Fatalf("round %d: split over two processes a send answered %v", round, err)
			}
		}
		if kept < 1 {
			t.Fatalf("round %d: no send was kept: %v", round, failures)
		}
		if held := voicesHeld(t, a, r.form.ID); held != 1 {
			t.Fatalf("round %d: %d voices held for one member across two processes, want 1", round, held)
		}
	}
}

func sqliteFilePair(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "forms.db")
	return sqliteFileDB(t, path), sqliteFileDB(t, path)
}

func TestUnderRefuseTwoProcessesSharingASqliteFileKeepOneVoice(t *testing.T) {
	first, second := sqliteFilePair(t)
	twoProcessVoiceRace(t, first, second, "twoprocs", models.OneVoiceRefuse)
}

func TestUnderFlagTwoProcessesSharingASqliteFileCountOneVoice(t *testing.T) {
	first, second := sqliteFilePair(t)
	twoProcessVoiceRace(t, first, second, "twoprocs", models.OneVoiceFlag)
}

func TestUnderRefuseTwoProcessesSharingAPostgresKeepOneVoice(t *testing.T) {
	twoProcessVoiceRace(t, postgresDBOrSkip(t), postgresDBOrSkip(t), "twoprocr", models.OneVoiceRefuse)
}

func TestUnderFlagTwoProcessesSharingAPostgresCountOneVoice(t *testing.T) {
	twoProcessVoiceRace(t, postgresDBOrSkip(t), postgresDBOrSkip(t), "twoprocf", models.OneVoiceFlag)
}
