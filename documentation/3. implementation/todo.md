# mwanachama-backend-forms (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| F4 | 🐞 **BUG — `CreateForm` clears a caller-supplied `id` (F1's own fix) but never clears `created_at`, so a caller can backdate a Form's own audit timestamp over real HTTP.** `form_impl.go`'s `Create` does `f.ID = ""` unconditionally, then separately does `if f.CreatedAt == "" { f.CreatedAt = models.NowRFC3339() }` — a non-empty caller value is honoured verbatim. `routes/form.go`'s `CreateForm` handler `readJSON`s straight into `mwanachamaforms.Form` (the same broad struct F1's own pin already exercises for `id`), so `created_at` is reachable the identical way. Demonstrated live through the real handler: `POST /forms` with body `{"created_at":"1999-01-01T00:00:00Z","title":"Chapter Census","originator_chapter_id":"chapter-1","closes_at":"<+1h>"}` returns **201** with `"created_at":"1999-01-01T00:00:00Z"` echoed back verbatim in the response, while `id` is correctly server-minted (a real UUID, confirming F1's own fix still holds — this is a distinct gap, not a F1 regression). Same shape independently found this run in `mwanachama-backend-actor` (filed there as ACT4, for `CreateActor`/`CreateGroup`'s `id`+`created_at`+`deleted`) and previously fixed fleet-wide in `mwanachama-backend-agency` as AG21/WK17 — this repo's own `Create` only partially applied that lesson (id only, not created_at). Not separately demonstrated but sharing the identical un-guarded `if t.CreatedAt == ""` idiom with no `t.ID = ""` clear at all: `target_impl.go`'s `AddTarget` — currently NOT reachable via HTTP for this vulnerability, since `routes/target.go`'s `AddTarget` handler decodes into a narrow inline struct (`chapter_id`/`includes_descendants` only) rather than the domain `Target` type directly, so a real caller cannot inject either field today; flagged as a latent Go-API-level risk for any future caller of `FormManager.AddTarget` directly, not a currently-exploitable HTTP gap. `public_impl.go`'s `Declare` has the same `if in.CreatedAt == ""` idiom on its create branch, but is confirmed NOT reachable via HTTP either — `routes/publiclink.go`'s `DeclareChapter` handler decodes into its own narrow inline struct (`respondent_id`/`declared_chapter_id`/`declared_text`) with no `id`/`created_at` field, so a real caller cannot inject either there. Fix location: `form_impl.go`'s `Create` — mint `CreatedAt` unconditionally, dropping the `if == ""` guard, the same fix ACT4/AG21 apply for the identical shape elsewhere in the fleet. | 📋 | Found 2026-09-23 by the fleet integration-test sweep, widening from this run's `mwanachama-backend-actor` ACT4 finding (loophole catalogue #1, "a key that does not identify the row" / caller-controlled server-owned field) into this repo, which shares actor's exact `Create<Type>` code shape by explicit design ("Structured exactly like mwanachama-backend-actor," this repo's own CLAUDE.md). Pinned by `routes/f4_createdat_forgery_test.go`'s `TestPinsF4_CreateFormHonoursCallerSuppliedCreatedAt`, committed and passing — must go RED once F4's fix lands. `go test ./...`/`go test -tags=integration ./...` both clean (no product code touched). |

_Nothing else open — see [todo_done.md](todo_done.md) for F1–F3 and the rest._

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
