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
