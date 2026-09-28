package mwanachamaforms_test

import (
	"context"
	"os"
	"testing"
	"time"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/postgres"
)

func newPostgresManager(t *testing.T) mwanachamaforms.FormManager {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test")
	}

	ctx := context.Background()
	sqlDB, err := postgres.Open(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("postgres.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	s, err := mwanachamaforms.SpecFor("formsi")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaforms.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			_ = db.Migrator().DropTable(s.TableFor(o))
		}
	})

	mgr, err := mwanachamaforms.NewFormManager(db, s)
	if err != nil {
		t.Fatalf("NewFormManager: %v", err)
	}
	return mgr
}

func TestPostgres_FormLifecycle_RoundTrip(t *testing.T) {
	mgr := newPostgresManager(t)
	ctx := context.Background()

	created, err := mgr.Create(ctx, mwanachamaforms.Form{
		Title: "Postgres round-trip", OriginatorChapterID: "chapter-1",
		ClosesAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := mgr.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Postgres round-trip" || got.Status != mwanachamaforms.StatusDraft {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	if _, err := mgr.AddTarget(ctx, mwanachamaforms.Target{FormID: created.ID, ChapterID: "chapter-2"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	out, _, err := mgr.Publish(ctx, created.ID, "", "admin")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if out.Status != mwanachamaforms.StatusOpen {
		t.Errorf("Publish result = %+v", out)
	}

	rollup, err := mgr.Rollup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Rollup: %v", err)
	}
	if rollup.Targeted != 1 || rollup.Stalled != 1 {
		t.Errorf("Rollup = %+v", rollup)
	}
}

func TestPostgres_AnswerConstraint_ExactlyOneValue(t *testing.T) {
	mgr := newPostgresManager(t)
	ctx := context.Background()

	f, err := mgr.Create(ctx, mwanachamaforms.Form{
		Title: "Postgres answer constraint", OriginatorChapterID: "chapter-1",
		ClosesAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := mgr.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{FormID: f.ID, AnswerType: mwanachamaforms.AnswerYesNo, Prompt: "Active?"}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := mgr.Publish(ctx, f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	yes := true
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "member-1", "chapter-2", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueBool: &yes}}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
}
