# mwanachama-backend-forms (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| F1 | 🐞 **BUG — `CreateForm` honors a caller-supplied `id`, and a duplicate id 500s instead of a clean conflict.** Driven through `routes.CreateForm`'s handler (sqlite, in-process; `routes/w_caller_supplied_id_test.go`): `POST /forms {"id":"attacker-chosen-form-id","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"<future>"}` returns `201` with `"id":"attacker-chosen-form-id"` — `form_impl.go`'s `Create` never clears `f.ID` before `gormstore.FormToRow(f)`, and `FormRow.BeforeCreate` only mints a UUID `if r.ID == ""`. Re-`POST`ing the identical body a second time hits the sqlite/Postgres primary-key unique constraint and 500s (`errors.go` has no duplicate-id sentinel — no `ErrDuplicateID`/`ErrConflict` equivalent — so `formStatusFor`'s default case answers an opaque `500`, never mapped to `409`). Same bug class independently found this same fleet sweep in `mwanachama-backend-taskmanager` (W11), `mwanachama-backend-assetmanager` (A12), `mwanachama-backend-accounting` (W15) and `mwanachama-backend-custody` (DEV-1695/1696) — already fixed properly in `mwanachama-backend-agency` (AG21) and handled safely by design in `mwanachama-backend-actor`/`mwanachama-backend-comm` (a caller-chosen id is allowed there too, but a collision maps cleanly to `ErrDuplicateID`/`ErrConflict` instead of a raw 500). **Fix location**: `form_impl.go`'s `Create`, clear `f.ID = ""` before building the row; separately, add a duplicate-id sentinel (actor's `classifyDuplicateID`/comm's `classify` is the template) mapped to 409 in `formStatusFor`. | 📋 | Found 2026-09-16 by the fleet integration-test sweep (loophole catalogue #3/#9). Two pinning tests committed in `routes/w_caller_supplied_id_test.go` (`TestCreateForm_PinsCallerSuppliedIDIsHonored`, `TestCreateForm_PinsDuplicateCallerSuppliedIDReturns500NotConflict`) asserting the *current* broken behavior. `go test ./...` and `go test -tags=integration ./...` both green with these added. This is the first numbered task on this board — no prior id scheme existed here, so `F1` starts one, matching the fleet's one-letter-per-repo convention (W=taskmanager/comm/accounting, A=assetmanager, G=git, ...). |

**The one prose line this board used to carry here** ("Wire
`mwanachama-backend-api-gateway`'s `internal/domain/survey.Repository`/
`.RegisterReader` to a new adapter type backed by `FormManager`... not
done here, by explicit scope decision") **turned out to be stale, not
open** — checked directly against the gateway's current code (2026-09-15)
before minting a board id for it, rather than trusting the text: no
`internal/domain/survey` package was ever created there.
`internal/store/formsadapter` declares its own `Repository`/
`RegisterReader` interfaces directly and already *is* the complete
wiring — `formsadapter.New(formManager, custodyLog)` is called in both
`cmd/server/stores.go` backends, gated on the live "forms" Module, backing
24+ registered routes across `survey_*_handlers.go`. The note was almost
certainly accurate when written (right after this repo's own extraction,
before the gateway-side wiring landed) and simply never updated once that
wiring shipped — the same drift this family's boards keep surfacing
elsewhere. No task minted; nothing left to do here.
