# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-forms

Extraction of `mwanachama-backend-api-gateway`'s `survey` domain (renamed
`Form` — the gateway's own domain, routes, and Postgres tables keep the word
`survey`; only this package's public naming changed) onto GORM: domain logic
AND storage both live in this package, imported directly by the gateway
process — no separate service, no gRPC, no proto. Module path
`github.com/aosanya/mwanachama-backend-forms`. Structured exactly like
`mwanachama-backend-actor` (its own extraction of the gateway's `member` and
`chapter` domains): same GORM-in-package pattern, same `models/`/`gormstore/`
split, same route-builder shape, same test-file layout.

Built 2026-09-04, at the user's request, following `mwanachama-backend-actor`
"exactly." No DSN-numbered decision record exists for this extraction in the
gateway's `documentation/2. design/todo.md` — unlike actor (DSN-1698), this
repo originates the decision rather than following one already recorded
there.

## Naming decision

The gateway's `internal/domain/survey.Survey` becomes `Form` here — the
user was asked whether to keep "Survey" (the word used everywhere else in
the product: requirements docs, mockups, todo acts) or rename to "Form"
(mirroring actor's own Member->Actor/Chapter->Group precedent, where the new
repo's name drives its own vocabulary) and chose the rename. Every
`SurveyID`/`survey_id` field across every type becomes `FormID`/`form_id`.
No other type was renamed — `Question`, `QuestionOption`, `Target`,
`Approval`, `Propagation`, `PublicLink`, `Respondent`, `Declaration`,
`Answer` keep their names, the same way actor only renamed its two root
nouns and left `Registration` (repackaged as `ActorGroupAssignment`, since it
*was* the Member-Chapter relation) as the only knock-on rename.

## Porting notes

- `manager.go`'s `FormManager` interface and `models/`'s domain types port
  the gateway's `internal/domain/survey` package's 32-method
  `Repository`/`RegisterReader` interface and 10 types field-for-field. The
  business logic itself was ported from the gateway's **memory** store
  (`internal/store/memory/survey_store*.go`), not its Postgres store — the
  memory implementation is Go logic close to this repo's own shape, where
  the Postgres store is SQL that would need re-deriving the same rules
  anyway.
- **Hand-formatted RFC3339 string timestamps, not GORM `time.Time`
  columns.** Every `time.Time`/`*time.Time` field in the gateway's survey
  types (there are many — `OpensAt`, `ClosesAt`, `PublishedAt`, ...) became a
  plain `string` here, `""` meaning unset — the same sort-safety reasoning
  actor's `models/time.go` documents, and the same "optional string, not
  pointer" convention actor already uses for `Group.ParentID`.
- **`Form` and `Target` gained `CreatedAt` fields they do not have upstream.**
  The gateway's `Survey`/`Target` rely on sequential Postgres ids
  (`"survey-1"`, `"st-1"`, ...) for `ORDER BY id` to mean "insertion order."
  This repo mints ids as UUIDs (actor's own storage convention, adopted here
  too), which carry no such order — so `ListForChapter` and `ListTargets`
  needed a real timestamp to sort by instead. `Target.CreatedAt` and
  `Form.CreatedAt`/`.LastUpdated` exist for exactly this reason, mirroring
  the same fix actor already applied to `Group`/`ActorGroupAssignment` when
  it made the identical UUID switch.
- **No cross-repo FK.** `Answer.MemberID`/`.ChapterID` FK to the gateway's
  own `member`/`chapter` Postgres tables upstream (migration 000057). Here
  they are plain, unconstrained string columns — mirroring actor's own
  documented choice for `Group.HierarchyID`/`.LevelID`/`.ParentID`
  ("no foreign key back to those tables").
- **No `Attributes`/`Property` catalog.** `attributes.go` and
  `models/property.go` exist in actor because `Actor`/`Group` needed a
  validated, extensible custom-field bag. Nothing in the survey/form domain
  is a free-form attribute bag — every field is fixed-shape — so this repo
  does not carry that machinery over. Skipping it is a deliberate deviation
  from copying actor file-for-file, in service of copying actor's *pattern*
  (build only the machinery a domain needs).
- **No custody/act-log writing.** The gateway's `AddOption` and
  `VersionQuestion` compose and write a custody `Entry` (an audit event) in
  the same call as the content write. `custody` is a gateway-internal domain
  this repo must not depend on — the same reasoning actor's
  `registration_impl.go` already gives for why `Deregister` writes no
  act-log row here. `models.AddOptionResult`/`.VersionQuestionResult` drop
  the gateway's `CustodyEventID`/`CustodyOccurredAt` fields accordingly; the
  gateway adapter, if one is ever written, is where that composition
  belongs, in its own separate call.
- **`ValidateAnswer`'s three gateway sentinels
  (`ErrAnswerShape`/`ErrAnswerTooLong`/`ErrUnknownOption`) collapse into
  one, `ErrInvalidAnswer`.** Every one of the three is caller input error
  either way (400 either way at the HTTP layer), so this repo uses the same
  one-sentinel-many-reasons shape `ErrInvalidActor` already uses for
  `models.ValidateAttributes` in actor, rather than threading three
  distinguishable sentinels through `models.ValidateAnswer`'s plain-error
  return for no behavioral gain.
- **`syncConstraints` (`gormstore/tables.go`), not
  `syncUniqueAttributeIndexes`.** Actor's GORM `Migrate` only ever needed
  postgres-dialect partial unique indexes for `Attributes` uniqueness. This
  domain's gateway schema also carries a Postgres trigger (migration
  000024's published-option-lock, refusing UPDATE/DELETE on a
  `QuestionOption` once its form leaves draft) and a multi-column CHECK
  (migration 000057's answer-has-exactly-one-value). `syncConstraints`
  applies all three (the trigger, the CHECK, and the approval
  one-open-per-form partial unique index) via raw SQL, postgres-only,
  skipped on sqlite — best-effort schema parity, not a certified 1:1 SQL
  port. `CollectionInterviewer` ports as a schema-valid enum value with its
  existing `ValidateAudienceCollection` rule; no interview-capture-specific
  behavior exists to port, upstream or here.
- **`PropagationKind`'s `Localized`/`Pushed` values and
  `Propagation.LastReminderAt` are inherited dead vocabulary, not a bug.**
  Confirmed via the gateway's own memory-store logic: no method anywhere
  transitions a row into `localized`/`pushed`, and nothing sets
  `LastReminderAt` — this repo carries the same schema-level parity without
  reproducing behavior that was never there to reproduce.
- **`mwanachama-backend-shared`, `-actor`, `-git`, `-taskmanager` were NOT
  touched** — this build was scoped to this repo only.
- **`mwanachama-backend-api-gateway` was NOT touched and is not wired to
  this repo** — no adapter satisfying its own `internal/domain/survey`
  package's interfaces was written here; that is a follow-up change, by
  explicit scope decision, the same as actor's own first build left the
  gateway's `stores.go`/`user_instances.go` wiring for later.

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- Route builder functions are named `"<ModelType>Routes"` exactly
  (`FormRoutes`, `QuestionRoutes`, `TargetRoutes`, `ApprovalRoutes`,
  `PropagationRoutes`, `PublicLinkRoutes`, `AnswerRoutes`) — see
  [routes/routes.go](routes/routes.go).
- Before wiring into `mwanachama-backend-api-gateway`, the interface that
  must NOT change is `internal/domain/survey.Repository`/`.RegisterReader` —
  route paths, request/response shapes and status codes all depend on that
  staying byte-for-byte identical. New adapter types satisfying it belong in
  the gateway's own `internal/store` tree, not here.
