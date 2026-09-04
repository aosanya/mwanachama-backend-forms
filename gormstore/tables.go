package gormstore

import (
	"fmt"

	"gorm.io/gorm"
)

// TableNames configures which physical tables a UserManager reads and
// writes — one per stored entity, mirroring
// mwanachama-backend-actor/gormstore.TableNames' shape for a domain with
// more entities.
type TableNames struct {
	Forms           string
	Questions       string
	QuestionOptions string
	Targets         string
	Approvals       string
	Propagation     string
	PublicLinks     string
	Respondents     string
	Declarations    string
	Answers         string
}

// DefaultTableNames builds the conventional table set for one mounted
// instance of this package, e.g. DefaultTableNames("forms") yields
// forms_forms, forms_questions, etc. — preserving the same
// multi-instance-mount capability mwanachama-backend-actor's
// DefaultTableNames provides.
func DefaultTableNames(instance string) TableNames {
	return TableNames{
		Forms:           instance + "_forms",
		Questions:       instance + "_questions",
		QuestionOptions: instance + "_question_options",
		Targets:         instance + "_targets",
		Approvals:       instance + "_approvals",
		Propagation:     instance + "_propagation",
		PublicLinks:     instance + "_public_links",
		Respondents:     instance + "_respondents",
		Declarations:    instance + "_declarations",
		Answers:         instance + "_answers",
	}
}

// Migrate creates or updates every table t names via GORM's AutoMigrate,
// scoped per table name, then applies [syncConstraints] for the handful of
// rules AutoMigrate cannot express (a trigger and two CHECK-shaped
// invariants) — postgres only, skipped on sqlite the same way
// mwanachama-backend-actor's syncUniqueAttributeIndexes skips non-
// postgres/sqlite dialects. Callers run this once at startup (or in test
// setup) before constructing a UserManager with the same db and t.
func Migrate(db *gorm.DB, t TableNames) error {
	migrations := []struct {
		table string
		row   any
	}{
		{t.Forms, &FormRow{}},
		{t.Questions, &QuestionRow{}},
		{t.QuestionOptions, &QuestionOptionRow{}},
		{t.Targets, &TargetRow{}},
		{t.Approvals, &ApprovalRow{}},
		{t.Propagation, &PropagationRow{}},
		{t.PublicLinks, &PublicLinkRow{}},
		{t.Respondents, &RespondentRow{}},
		{t.Declarations, &DeclarationRow{}},
		{t.Answers, &AnswerRow{}},
	}
	for _, m := range migrations {
		if err := db.Table(m.table).AutoMigrate(m.row); err != nil {
			return fmt.Errorf("Migrate: %s: %w", m.table, err)
		}
	}
	return syncConstraints(db, t)
}

// syncConstraints applies the rules AutoMigrate's row-tag mapping cannot
// express: the published-option-lock trigger (gateway migration 000024),
// the answer exactly-one-value CHECK, and the approval one-open-per-form
// partial unique index. Postgres only — sqlite (this repo's unit tests)
// runs without them, the same tradeoff mwanachama-backend-actor already
// accepts for its own postgres-only unique-attribute indexes. Best-effort
// schema parity, not a certified 1:1 port of the gateway's SQL — see this
// repo's CLAUDE.md gap note.
func syncConstraints(db *gorm.DB, t TableNames) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}

	lockFn := t.QuestionOptions + "_locked"
	lockTrigger := t.QuestionOptions + "_lock_trigger"
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
$$ LANGUAGE plpgsql`, lockFn, t.Questions, t.Forms),
		fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s`, lockTrigger, t.QuestionOptions),
		fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s()`,
			lockTrigger, t.QuestionOptions, lockFn),

		fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s_has_exactly_one_value`, t.Answers, t.Answers),
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s_has_exactly_one_value CHECK (
	(CASE WHEN option_ids IS NOT NULL AND option_ids::text NOT IN ('[]', 'null') THEN 1 ELSE 0 END) +
	(CASE WHEN value_text <> '' THEN 1 ELSE 0 END) +
	(CASE WHEN value_number IS NOT NULL THEN 1 ELSE 0 END) +
	(CASE WHEN value_date <> '' THEN 1 ELSE 0 END) +
	(CASE WHEN value_time <> '' THEN 1 ELSE 0 END) +
	(CASE WHEN value_bool IS NOT NULL THEN 1 ELSE 0 END) = 1
)`, t.Answers, t.Answers),

		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS %s_one_open_idx ON %s (form_id) WHERE decided_at = ''`,
			t.Approvals, t.Approvals),
	}
	for _, sql := range stmts {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("syncConstraints: %w", err)
		}
	}
	return nil
}
