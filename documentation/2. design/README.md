# Design

GORM-backed relational storage — domain types in `../../models/`, row
structs and migration in `../../gormstore/`. No entity-graph, no generic
runtime schema: fixed Go structs mapped to plain tables, the same choice
`mwanachama-backend-actor` made (`gormstore/tables.go`'s `Migrate`).

## Tables (one per stored entity, `gormstore.DefaultTableNames(instance)`)

`<instance>_forms`, `_questions`, `_question_options`, `_targets`,
`_approvals`, `_propagation`, `_public_links`, `_respondents`,
`_declarations`, `_answers`.

## Topology

```
Form ──has──────────► Question ──has──────► QuestionOption
Form ──has──────────► Target        (member-audience forms only)
Form ──has──────────► Approval      (one open at a time)
Form ──has──────────► Propagation   (one per Target, after Publish)
Form ──has──────────► PublicLink    (public-audience forms only, one, after Publish)
Respondent ──answers──► Answer ──answers──► Question
Respondent ──declares─► Declaration ──about──► Form
```

## Decisions carried in from mwanachama-backend-actor's precedent

- GORM-in-package, hand-formatted RFC3339 string timestamps (not GORM
  `time.Time` columns), UUID row ids via `BeforeCreate`, no GORM
  associations across mounted-instance table names, no cross-repo foreign
  keys (`Answer.MemberID`/`.ChapterID` are plain columns).
- `Form.CreatedAt`/`.LastUpdated` and `Target.CreatedAt` exist purely so a
  list has something meaningful to sort by now that ids are UUIDs, not
  sequential — see `CLAUDE.md`.

## Constraints AutoMigrate can express directly (GORM tags)

Unique on `(form_id, chapter_id)` for `Target` and `Propagation`; unique on
`key` for `PublicLink`; unique on `public_key` for `Respondent`; unique on
`(respondent_id, form_id)` for `Declaration`; unique on `(question_id,
member_id)` for `Answer`.

## Constraints applied via `gormstore.syncConstraints` (raw SQL, postgres only)

- The published-option-lock trigger (refuses `UPDATE`/`DELETE` on a
  `QuestionOption` row once its form leaves `draft`).
- `Answer`'s exactly-one-value CHECK across `option_ids`/`value_text`/
  `value_number`/`value_date`/`value_time`/`value_bool`.
- `Approval`'s one-open-per-form partial unique index (`WHERE decided_at =
  ''`).

Skipped on sqlite — this repo's unit tests run without them, the same
tradeoff actor already accepts for its own postgres-only unique-attribute
indexes.

## Not carried over

No `Attributes`/`Property` catalog (nothing here is a free-form attribute
bag), no custody/act-log writes on `AddOption`/`VersionQuestion` (this repo
does not depend on the gateway's custody domain). See `CLAUDE.md` for the
full account.
