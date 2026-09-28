# mwanachama-backend-forms — completed work

## Initial build — 2026-09-04

Extracted `mwanachama-backend-api-gateway`'s `survey` domain onto GORM,
structured exactly like `mwanachama-backend-actor`'s own extraction of
`member`/`chapter` (GORM-in-package, `models/`/`gormstore/` split, route
builders named `"<ModelType>Routes"`, sqlite-in-memory unit tests plus an
opt-in `POSTGRES_URL`-gated integration test). Renamed the gateway's root
noun `Survey` to `Form` (and every `SurveyID` field to `FormID`) — a
decision the user made explicitly when asked, mirroring actor's own
Member→Actor/Chapter→Group precedent.

Built: `doc.go`, `tables.go`, `errors.go`, `manager.go` (`FormManager`
interface), `form_impl.go`, `approval_impl.go`, `question_impl.go`,
`target_impl.go`, `propagation_impl.go`, `public_impl.go`,
`register_impl.go`; `models/` (`form.go`, `question.go`, `target.go`,
`approval.go`, `propagation.go`, `public.go`, `answer.go`, `register.go`,
`time.go`); `gormstore/` (one row file per entity plus `tables.go`'s
`Migrate`/`syncConstraints`); `routes/` (`routes.go`, `wire.go`, `doc.go`,
one handler file per entity, `routes_test.go`); the four-phase
`documentation/` layout and this repo's `CLAUDE.md`.

Business logic (every status-transition guard, validation rule, and
computed field across the gateway's 32-method `Repository` interface) was
extracted from the gateway's `internal/store/memory/survey_store*.go` — the
Go reference implementation, closer to this repo's own shape than
translating the Postgres SQL store would have been — and verified against
this repo's own `go test ./...` (sqlite in-memory), not a real Postgres
container, per this org's standing test-verification convention.

Deliberately not carried over from the gateway: the `Attributes`/`Property`
catalog machinery (nothing in this domain is a free-form attribute bag), and
custody/act-log writes on `AddOption`/`VersionQuestion` (this repo does not
depend on the gateway's custody domain) — both considered exclusions, not
oversights; see `CLAUDE.md` for the full account of every deviation from a
literal file-for-file copy of `mwanachama-backend-actor`.

Not done, by explicit scope decision: wiring this package into
`mwanachama-backend-api-gateway` itself (a new adapter satisfying
`internal/domain/survey.Repository`/`.RegisterReader`) — see `todo.md`.

## Gateway wiring — actually already done, corrected 2026-09-15

The line immediately above (and `todo.md`'s matching prose) was stale, not
an open task — caught while `developer`'s compiled cross-repo backlog
pointed at it as "no board id exists for this." Checked against
`mwanachama-backend-api-gateway`'s current code before minting a row for
it: no `internal/domain/survey` package was ever created there. The
gateway's `internal/store/formsadapter` (built as part of this repo's own
2026-09-04 initial-build session, see above) declares its own
`Repository`/`RegisterReader` interfaces directly and already *is* the
complete wiring — `formsadapter.New(formManager, custodyLog)` is
constructed in both `cmd/server/stores.go` backends, gated on the live
"forms" Module (`cmd/server/modules.go`/`router.go`), backing 24+
registered routes across that repo's `survey_*_handlers.go` files. The
note above was almost certainly accurate the moment it was written — right
after this repo's own extraction, before the gateway-side wiring had
happened yet — and simply never got updated once that wiring shipped.
`todo.md` corrected to "nothing open" instead of minting a task for
already-done work.

## F1 — `CreateForm` ignores a caller-supplied `id` — 2026-09-21

`form_impl.go`'s `Create` now clears `f.ID` before building the row, so `FormRow.BeforeCreate` always mints the UUID; re-posting the same body creates a second distinct Form rather than a primary-key 500. The row also proposed a duplicate-id sentinel mapped to 409 in `formStatusFor`; not added, because with the id server-owned a collision can no longer come from caller input. The two pins in `routes/w_caller_supplied_id_test.go` went red as designed and were rewritten to assert the fixed behaviour. Verified: `go build ./... && go vet ./... && go test ./...` and `go test -tags=integration ./...` green, `gofmt -l .` clean. Done with taskmanager W11 and assetmanager A12.

## F2 — `Declare` rejects a respondent that does not exist — 2026-09-21

`public_impl.go`'s `Declare` now counts `Respondents` rows with the given id first and returns `ErrInvalidReference` (already mapped to 400 in `formStatusFor`, and documented as "a write names an id that does not exist") when there is none, instead of upserting a Declaration for a phantom id. Fixed in the manager rather than only the `DeclareChapter` handler so every caller gets it. No new sentinel. Not done: the check proves the respondent exists, not that it belongs to this form's link — `models/public.go` documents that a respondent key is deliberately not bound to one form, so binding it would be a design change. `routes/f2_phantom_declaration_test.go` now asserts a phantom id gets 400 and a respondent minted through `UpsertRespondent` still gets 201. Verified: `go build ./... && go vet ./... && go test ./...` and `go test -tags=integration ./...` green, `gofmt -l .` clean.

## F3 — `Publish` no longer strands a Form `open` when its PublicLink insert fails — 2026-09-21

`form_impl.go`'s `Publish` now runs the Form status update and the PublicLink insert (or, for member-audience forms, the per-Target Propagation inserts, which had the same partial-write shape) in one transaction, so any failure rolls the status back and the Form stays publishable. Added `ErrLinkKeyTaken` (409 in `routes/wire.go`) and a pre-check for the candidate key inside the transaction, so the common collision gives a clean conflict rather than an opaque 500; a true race between two publishers still hits the `PublicLinkRow.Key` unique index and rolls back, surfacing as a 500 with no stranded state. `Publish`'s doc comment now says the key is checked. `routes/f3_publish_key_collision_test.go` asserts 409, the Form not left `open`, and a retry with a fresh key minting a real link. Verified: `go build ./... && go vet ./... && go test ./...` and `go test -tags=integration ./...` green, `gofmt -l .` clean.

## F6–F12 — the objects are declared, not written — 2026-09-28

The declared-domain conversion, decided with the owner and recorded as
[FMD-001](../2.%20design/todo_details/FMD-001.md), which is where the
reasoning lives rather than here. `mwanachama-backend-catalog` was the
reference shape at the owner's direction; `mwanachama-backend-agency`'s
AGD-007 was the worked precedent for `Provision`.

**F6 — scope decided.** Complete (storage and HTTP; there is no MCP surface
here). Tables become `<instance>_forms_<object>`, the instance a mount's
choice through `SpecFor`. F5's form→record mapping is **not** declared yet —
it is still an open design, and guessing its shape now would be worse than a
later blueprint edit. The trigger, the CHECK and the one-open-approval index
move into `Provision` as raw SQL, Postgres-only, because none of the three is
expressible in the format. This repo had no design board; it has one now
(`FMD-XXX`).

**Three of the plan's premises were wrong, and are corrected in FMD-001.**
The live tables were never called `survey*` (they have been `form_*` since
the 2026-09-05 cutover, and migration 000003 was rewritten then to match);
this repo's `routes/` package had no consumer outside its own tests; and the
consumer that matters is `mwanachama-wakala-api`, not the gateway — owner's
direction mid-conversion, which is what F11 actually shipped against.

**F7 — `forms.blueprint.json`.** Ten objects (the board said nine;
`Respondent` and `Declaration` are two), every field and index declared with
prose on each. The RFC3339 string timestamps are the format's `timestamp`
type; `Status`, `Audience`, `CollectionMode`, `AnswerType`, `Decision`,
`PropagationKind` and `LinkStatus` are declared enums with their values.

**F8 — `forms.forms.json` and `Provision`.** The domain spec ships `wakala`
as its example instance. `Provision` renames `<instance>_<object>` to
`<instance>_forms_<object>`, carries `updated_at` across to `last_updated`
(the store joins a column to a Go field by name, so leaving it would read
every timestamp back empty), drops the old `idx_`-prefixed indexes, runs
`spec.Migrate`, then applies the three raw-SQL rules. It refuses rather than
proceeds if both the legacy and declared tables exist and the legacy one
still holds rows.

**F9 — the spec-driven store.** `gormstore/` and root `tables.go` deleted:
ten row structs, their `XToRow`/`XFromRow` pairs, `Migrate`,
`syncConstraints`, `TableNames` and `DefaultTableNames`. `store.go` holds the
role constants and the codec wrappers; `manager.go` holds `q`/`take`/`find`/
`listOf`/`insert`/`replace` and their transaction variants;
`NewFormManager(db, *spec.Spec)` replaces the old constructor and checks the
spec against the carried types. `validate.go` applies the declared `required`
and enum rules on the way in, with `Check` for a caller with no database yet.
**F4 is closed here**: `Create` mints `created_at` unconditionally, and
`AddTarget` now clears `id` and mints `created_at` too — the latent half F4's
own notes flagged as unreachable over HTTP, which the widened request shape
would otherwise have opened. Its pin inverted into
`routes/f4_createdat_regression_test.go`'s
`TestCreateFormMintsItsOwnCreatedAt` rather than being deleted.

**F10 — `forms.operations.json`.** Thirty operations replace ten route files;
`routes/` is a 100-line adapter over `dispatch`. Every address now carries a
gating action, and `AnonymousActions` is three: opening a shareable link,
registering a respondent behind one, declaring a chapter behind one.
Answering is deliberately **not** anonymous — this module cannot tell a
public respondent from a member. Two signature changes the binder forced, both
recorded in FMD-001: `AddQuestion`/`UpdateQuestion`/`VersionQuestion` take one
`models.QuestionDraft`, and the public-link surface became three manager
methods (`OpenPublicLink`, `RegisterRespondent`, `DeclareChapter`) because the
gate they perform is domain logic, not argument binding. `actor_id` now comes
`from: caller`; neither method used it, so that cost nothing. The `ErrInvalidAnswer`
sentinel F10 predicted might reach the wire as a bare 500 was already mapped —
the two genuinely new ones are `ErrInvalidForm`/`ErrInvalidQuestion`, from the
declared rules, both mapped to 400.

**F11 — the consumers.** `mwanachama-wakala-api` gained a real mount, built
the way its catalog mount is: `cmd/server/forms.go` (env-gated on
`FORMS_DATABASE_URL`/`FORMS_INSTANCE`, unset leaves it unmounted),
`internal/api/http/forms_routes.go`, a `Forms` field on `Deps`, the
`module:forms` scope, and `internal/api/http/forms_routes_test.go` covering
the gate, a granted caller, the anonymous link surface, answering *not* being
anonymous, and an unmounted module registering nothing.
`mwanachama-backend-api-gateway` was held to still building and testing
green: `formsadapter` updated for the three signatures (new `toModelDraft`),
`stores.go`'s two forms blocks and the two backend tests moved to
`SpecFor`/`Provision`. Its Postman gate is still red at 28 uncovered and 16
orphans, every one of them `/v1/agency/*` (DEV-1693/DEV-1694) — this
conversion moved neither number. `cmd/backfill-form-gorm` was moved to
`dump/mwanachama-backend-api-gateway/cmd/` rather than deleted: it is built
entirely from the row structs F9 removes, and no migration creates the
`survey_*` tables it read from any more.

**F12 — the documentation.** CLAUDE.md rewritten — it told the next session
to follow `mwanachama-backend-actor`'s `models/`+`gormstore/`+route-builder
shape "exactly", which the conversion makes the opposite of true. Root `doc.go`
deleted outright rather than rewritten — a package doc is still a doc
comment, and what it held now lives in `documentation/2. design/`. The
design board, FMD-001 and `storage.md` created, the stale design README
rewritten, and the pointers to shared's `declared-domains.md`/`dispatcher.md`
added.

**A full comment strip followed**, at the owner's direction: every comment
in every Go file in this repo is gone, `//go:` directives and build tags
aside. That is the rule this repo's own CLAUDE.md carries and the
conversion had been adding new comments against it. What was load-bearing
moved into `documentation/2. design/storage.md` — chiefly why `TimeLayout`
fixes nine fractional digits, which neither `time.RFC3339` nor
`time.RFC3339Nano` gets right for a text column that is sorted as text.

**The engine needed extending first**, which the owner chose over flattening
this module's types: `mwanachama-backend-shared`'s `spec` gained a `float`
type and a per-field `nullable` flag, and `specstore` gained pointer, float
and JSON-collection arms plus a construction-time refusal of a carrier that
cannot hold the distinction. A real bug surfaced with it — `ColumnName`
derived `option_i_ds` from `OptionIDs` — now fixed and pinned. Full record in
FMD-001. `mwanachama-backend-comm`'s concurrent conversion is the second
consumer of that work.

Verified: `go build ./... && go vet ./... && go test ./... && gofmt -l .`
clean in `mwanachama-backend-forms`, `mwanachama-backend-shared` and
`mwanachama-wakala-api`. `mwanachama-backend-api-gateway`,
`mwanachama-backend-catalog`, `mwanachama-backend-agency` and
`mwanachama-backend-permissions` all rebuild, with only their own
pre-existing reds (the two Postman gates, and the sweep's deliberately-red
`_OpenHole_` tests CAT13/CAT14, AG48, PM6). Not run: this repo's
`postgres_integration_test.go`, which is `POSTGRES_URL`-gated and not part
of the default bar — so the three raw-SQL rules `Provision` applies are
compiled and unchanged in substance, but not re-executed against a real
Postgres this pass.
