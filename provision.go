package mwanachamaforms

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

// Provision brings a database up to what s declares: it moves a pre-spec
// table set onto the declared names, creates whatever is missing, and
// applies the rules a spec has no way to state.
func Provision(db *gorm.DB, s *spec.Spec) error {
	if err := renameLegacy(db, s); err != nil {
		return err
	}
	if err := spec.Migrate(db, s); err != nil {
		return err
	}
	return syncConstraints(db, s)
}

// legacyTable is what this object's table was called before the module
// segment existed: <instance>_<table>, where the declared name is
// <instance>_<module>_<table>.
func legacyTable(s *spec.Spec, o spec.Object) string {
	return s.Instance + "_" + o.Table
}

const (
	legacyTimestampColumn = "updated_at"
	timestampColumn       = "last_updated"
)

func renameLegacy(db *gorm.DB, s *spec.Spec) error {
	m := db.Migrator()
	for _, o := range s.Objects {
		declared := s.TableFor(o)
		legacy := legacyTable(s, o)

		if legacy != declared && m.HasTable(legacy) {
			if m.HasTable(declared) {
				stranded, err := rowCount(db, legacy)
				if err != nil {
					return err
				}
				if stranded > 0 {
					return fmt.Errorf("forms: %s and %s both exist and %s still holds %d row(s), which no read would ever reach again",
						legacy, declared, legacy, stranded)
				}
				continue
			}
			if err := db.Exec(fmt.Sprintf("alter table %s rename to %s", legacy, declared)).Error; err != nil {
				return fmt.Errorf("forms: rename %s to %s: %w", legacy, declared, err)
			}
			if err := dropLegacyIndexes(db, declared); err != nil {
				return err
			}
		}

		if err := renameTimestamp(db, s, o); err != nil {
			return err
		}
	}
	return nil
}

// renameTimestamp carries a pre-spec row's updated_at across to the column
// the declared field name derives, which is last_updated. The store joins a
// declared column to a Go field by name, so leaving the old one behind would
// read every timestamp back empty rather than fail.
func renameTimestamp(db *gorm.DB, s *spec.Spec, o spec.Object) error {
	declared := s.TableFor(o)
	if !db.Migrator().HasTable(declared) || !declaresColumn(o, timestampColumn) {
		return nil
	}
	legacy, err := hasColumn(db, declared, legacyTimestampColumn)
	if err != nil {
		return err
	}
	current, err := hasColumn(db, declared, timestampColumn)
	if err != nil {
		return err
	}
	if !legacy || current {
		return nil
	}
	stmt := fmt.Sprintf("alter table %s rename column %s to %s",
		declared, legacyTimestampColumn, timestampColumn)
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("forms: rename %s.%s: %w", declared, legacyTimestampColumn, err)
	}
	return nil
}

func dropLegacyIndexes(db *gorm.DB, table string) error {
	query := `select name from sqlite_master where type = 'index' and tbl_name = ?`
	if db.Dialector.Name() == "postgres" {
		query = `select indexname from pg_indexes where tablename = ?`
	}
	var names []string
	if err := db.Raw(query, table).Scan(&names).Error; err != nil {
		return fmt.Errorf("forms: read indexes of %s: %w", table, err)
	}
	for _, name := range names {
		if len(name) < 4 || name[:4] != "idx_" {
			continue
		}
		if err := db.Exec("drop index if exists " + name).Error; err != nil {
			return fmt.Errorf("forms: drop index %s: %w", name, err)
		}
	}
	return nil
}

func rowCount(db *gorm.DB, table string) (int64, error) {
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("forms: count %s: %w", table, err)
	}
	return n, nil
}

func hasColumn(db *gorm.DB, table, column string) (bool, error) {
	query := `select 1 from pragma_table_info(?) where name = ?`
	if db.Dialector.Name() == "postgres" {
		query = `select 1 from information_schema.columns where table_name = ? and column_name = ?`
	}
	var found []int
	if err := db.Raw(query, table, column).Scan(&found).Error; err != nil {
		return false, fmt.Errorf("forms: read columns of %s: %w", table, err)
	}
	return len(found) > 0, nil
}

func declaresColumn(o spec.Object, column string) bool {
	for _, f := range o.Fields {
		if f.Name == column {
			return true
		}
	}
	return false
}

func tableFor(s *spec.Spec, role string) string {
	o, ok := s.ByRole(role)
	if !ok {
		return ""
	}
	return s.TableFor(o)
}

// syncConstraints applies the three rules the declaration has no way to
// carry: an option may not move once its form has left draft, an answer
// fills exactly one value column, and a form has at most one undecided
// approval. The first two have no declared equivalent at all; the third is a
// partial index over a condition that is not `deleted`, which is the only
// one the format names.
//
// Postgres only, as it was before the conversion — SQLite runs the unit
// tests without them, and the Go rules that also enforce all three are what
// those tests exercise.
func syncConstraints(db *gorm.DB, s *spec.Spec) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}

	forms := tableFor(s, roleForm)
	questions := tableFor(s, roleQuestion)
	options := tableFor(s, roleQuestionOption)
	answers := tableFor(s, roleAnswer)
	approvals := tableFor(s, roleApproval)

	lockFn := options + "_locked"
	lockTrigger := options + "_lock_trigger"
	stmts := []string{
		fmt.Sprintf(`CREATE OR REPLACE FUNCTION %s() RETURNS trigger AS $$
DECLARE
	form_status text;
BEGIN
	SELECT f.status INTO form_status
	FROM %s q JOIN %s f ON f.id = q.form_id
	WHERE q.id = COALESCE(NEW.question_id, OLD.question_id);
	IF form_status IS DISTINCT FROM 'draft' THEN
		RAISE EXCEPTION 'cannot modify options once the form has left draft';
	END IF;
	RETURN OLD;
END;
$$ LANGUAGE plpgsql`, lockFn, questions, forms),
		fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s`, lockTrigger, options),
		fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s()`,
			lockTrigger, options, lockFn),

		fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s_has_exactly_one_value`, answers, answers),
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s_has_exactly_one_value CHECK (
	(CASE WHEN option_ids IS NOT NULL AND option_ids::text NOT IN ('[]', 'null') THEN 1 ELSE 0 END) +
	(CASE WHEN value_text <> '' THEN 1 ELSE 0 END) +
	(CASE WHEN value_number IS NOT NULL THEN 1 ELSE 0 END) +
	(CASE WHEN value_date <> '' THEN 1 ELSE 0 END) +
	(CASE WHEN value_time <> '' THEN 1 ELSE 0 END) +
	(CASE WHEN value_bool IS NOT NULL THEN 1 ELSE 0 END) = 1
)`, answers, answers),

		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS %s_one_open_idx ON %s (form_id) WHERE decided_at = ''`,
			approvals, approvals),
	}
	for _, sql := range stmts {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("forms: syncConstraints: %w", err)
		}
	}
	return nil
}
