# mwanachama-backend-forms (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

Nothing open — see [todo_done.md](todo_done.md).

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
