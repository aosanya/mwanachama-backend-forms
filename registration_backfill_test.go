package mwanachamaforms_test

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	declaredAt   = "2026-02-01T09:00:00.000000000Z"
	registeredAt = "2026-01-15T08:00:00.000000000Z"
	firstSeenAt  = "2026-01-10T07:00:00.000000000Z"
)

type backfillBackend struct {
	name string
	open func(t *testing.T) (*gorm.DB, *spec.Spec)
}

func backfillBackends() []backfillBackend {
	return []backfillBackend{
		{"sqlite", func(t *testing.T) (*gorm.DB, *spec.Spec) { return clashDB(t) }},
		{"postgres", func(t *testing.T) (*gorm.DB, *spec.Spec) {
			s, err := forms.SpecFor(clashSlug)
			if err != nil {
				t.Fatalf("SpecFor: %v", err)
			}
			return localPostgresSchema(t), s
		}},
	}
}

func localPostgresSchema(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Skip("DATABASE_URL does not point at 127.0.0.1 or localhost")
	}
	quiet := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	admin, err := gorm.Open(postgres.Open(dsn), quiet)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	schema := fmt.Sprintf("backfill_%d", time.Now().UnixNano())
	if err := admin.Exec("create schema " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), quiet)
	if err != nil {
		t.Fatalf("open postgres in %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		admin.Exec("drop schema " + schema + " cascade")
		if sqlDB, err := admin.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func eachBackend(t *testing.T, run func(t *testing.T, db *gorm.DB, s *spec.Spec)) {
	t.Helper()
	for _, b := range backfillBackends() {
		t.Run(b.name, func(t *testing.T) {
			db, s := b.open(t)
			run(t, db, s)
		})
	}
}

func addRespondent(t *testing.T, db *gorm.DB, s *spec.Spec, id string) {
	t.Helper()
	stmt := "insert into " + tableOf(t, s, "respondent") + " (id, public_key, first_seen_at) values (?, ?, ?)"
	if err := db.Exec(stmt, id, "key-"+id, firstSeenAt).Error; err != nil {
		t.Fatalf("insert respondent %s: %v", id, err)
	}
}

func addDeclaration(t *testing.T, db *gorm.DB, s *spec.Spec, respondent, form string) {
	t.Helper()
	stmt := "insert into " + tableOf(t, s, "declaration") + " (id, respondent_id, form_id, declared_text, created_at) values (?, ?, ?, 'ward 3', ?)"
	if err := db.Exec(stmt, "dec-"+respondent+"-"+form, respondent, form, declaredAt).Error; err != nil {
		t.Fatalf("insert declaration %s/%s: %v", respondent, form, err)
	}
}

func addRegistration(t *testing.T, db *gorm.DB, s *spec.Spec, respondent, form string) {
	t.Helper()
	stmt := "insert into " + tableOf(t, s, "registration") + " (id, respondent_id, form_id, registered_at) values (?, ?, ?, ?)"
	if err := db.Exec(stmt, "reg-"+respondent+"-"+form, respondent, form, registeredAt).Error; err != nil {
		t.Fatalf("insert registration %s/%s: %v", respondent, form, err)
	}
}

type registrationRow struct {
	RespondentID string
	FormID       string
	RegisteredAt string
}

func registrationsHeld(t *testing.T, db *gorm.DB, s *spec.Spec) map[string]registrationRow {
	t.Helper()
	var rows []registrationRow
	query := "select respondent_id, form_id, registered_at from " + tableOf(t, s, "registration")
	if err := db.Raw(query).Scan(&rows).Error; err != nil {
		t.Fatalf("read registrations: %v", err)
	}
	held := make(map[string]registrationRow, len(rows))
	for _, r := range rows {
		held[r.RespondentID+"/"+r.FormID] = r
	}
	return held
}

func theStateBeforeRegistrationWasChecked(t *testing.T, db *gorm.DB, s *spec.Spec) {
	t.Helper()
	mustProvision(t, db, s)
	addRespondent(t, db, s, "r-both")
	addRespondent(t, db, s, "r-one")
	addRespondent(t, db, s, "r-kept")
	addRespondent(t, db, s, "r-silent")
	addDeclaration(t, db, s, "r-both", "f-1")
	addDeclaration(t, db, s, "r-both", "f-2")
	addDeclaration(t, db, s, "r-one", "f-1")
	addDeclaration(t, db, s, "r-kept", "f-1")
	addRegistration(t, db, s, "r-kept", "f-1")
	addDeclaration(t, db, s, "r-ghost", "f-1")
}

func TestARespondentWhoDeclaredBeforeRegistrationWasCheckedIsRegisteredOnTheFirstFormsCallAfterStart(t *testing.T) {
	eachBackend(t, func(t *testing.T, db *gorm.DB, s *spec.Spec) {
		theStateBeforeRegistrationWasChecked(t, db, s)

		mustProvision(t, db, s)

		held := registrationsHeld(t, db, s)
		for _, pair := range []string{"r-both/f-1", "r-both/f-2", "r-one/f-1"} {
			if _, ok := held[pair]; !ok {
				t.Errorf("no registration for %s after the first forms call, want one for each link already declared on", pair)
			}
		}
		if len(held) != 4 {
			t.Errorf("%d registrations held, want 4: the three declared pairs and the one already there; got %v", len(held), held)
		}
	})
}

func TestTheBackfillRegistersOnlyWhatWasDeclaredAndNeverWithoutARespondent(t *testing.T) {
	eachBackend(t, func(t *testing.T, db *gorm.DB, s *spec.Spec) {
		theStateBeforeRegistrationWasChecked(t, db, s)

		mustProvision(t, db, s)

		held := registrationsHeld(t, db, s)
		if _, ok := held["r-silent/f-1"]; ok {
			t.Errorf("a respondent who declared nowhere was given a registration")
		}
		if _, ok := held["r-ghost/f-1"]; ok {
			t.Errorf("a declaration whose respondent no longer exists was given a registration")
		}
		if got := held["r-kept/f-1"].RegisteredAt; got != registeredAt {
			t.Errorf("an existing registration reads registered_at %q, want it left at %q", got, registeredAt)
		}
		if got := held["r-both/f-1"].RegisteredAt; got != declaredAt {
			t.Errorf("a backfilled registration reads registered_at %q, want the time of the declaration %q", got, declaredAt)
		}
	})
}

func TestTheBackfillCountsWhatItChangedAndAnotherRunChangesNothing(t *testing.T) {
	eachBackend(t, func(t *testing.T, db *gorm.DB, s *spec.Spec) {
		theStateBeforeRegistrationWasChecked(t, db, s)

		first, err := forms.BackfillRegistrations(db, s)
		if err != nil {
			t.Fatalf("BackfillRegistrations: %v", err)
		}
		again, err := forms.BackfillRegistrations(db, s)
		if err != nil {
			t.Fatalf("BackfillRegistrations again: %v", err)
		}

		if first.Registrations != 3 || first.Respondents != 2 {
			t.Errorf("first run counts %+v, want 3 registrations for 2 respondents", first)
		}
		if again.Registrations != 0 || again.Respondents != 0 {
			t.Errorf("second run counts %+v, want nothing changed", again)
		}
		if len(registrationsHeld(t, db, s)) != 4 {
			t.Errorf("registrations after two runs: %v, want 4", registrationsHeld(t, db, s))
		}
	})
}

func TestTwoProcessesBackfillingTogetherRegisterEveryPairOnceAndNeitherFails(t *testing.T) {
	db, s := backfillBackends()[1].open(t)
	theStateBeforeRegistrationWasChecked(t, db, s)
	second := localPostgresSharing(t, db)

	var wg sync.WaitGroup
	counts := make([]forms.RegistrationsBackfilled, 2)
	errs := make([]error, 2)
	for i, handle := range []*gorm.DB{db, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counts[i], errs[i] = forms.BackfillRegistrations(handle, s)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("process %d failed: %v", i, err)
		}
	}
	if total := counts[0].Registrations + counts[1].Registrations; total != 3 {
		t.Errorf("the two processes changed %d registrations between them, want 3", total)
	}
	if len(registrationsHeld(t, db, s)) != 4 {
		t.Errorf("registrations held: %v, want 4", registrationsHeld(t, db, s))
	}
}

func localPostgresSharing(t *testing.T, first *gorm.DB) *gorm.DB {
	t.Helper()
	var schema string
	if err := first.Raw("select current_schema()").Scan(&schema).Error; err != nil {
		t.Fatalf("current schema: %v", err)
	}
	u, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open a second process on %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}
