# mwanachama-backend-forms (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| F5 | ⚠️ **Architecture gap — forms is meant to be the input designer only, but `Answer` is the terminal store, so collected data has nowhere to land.** Owner's direction, 2026-09-28: a form designs the instrument; the data it collects belongs in a domain store the way a contribution belongs in `mwanachama-backend-accounting`'s ledger rather than in whatever captured it. Today there is no such seam. `models.Answer` is a row against a `Question` and that is where a response stops: nothing on `Form` or `Question` names a target module, a record type or a field, and no write path hands a closed round to another package. Four concrete consequences, all met while authoring the `party-mobilization` agency template, whose whole sentiment goal rests on this: (1) **no target mapping** — "voting intention" is an answer to question 3 of form 7, not a field on a record anything else can read; (2) **no subject key** — `Answer` carries no reference to the person answering, so a response cannot be tied to a member without re-asking for an identifier as a question; (3) **no collection context** — `models.Target` scopes who a form goes *to*, not where an answer came *from*, so geography (the ward, in that template's case) has to be asked as a question and re-derived afterwards; (4) **no recurrence** — `Form` carries one `OpensAt`/`ClosesAt` pair, so a weekly or monthly return is a separate Form each period and comparison across periods is manual. Shape to design: a mapping declared on the Form (`question → target module, record type, field`) plus a "land a closed round" operation that writes those records into the target module and reconciles the landed count against the collected count, refusing to field an instrument whose questions are not all mapped. Precedent to copy: accounting closes `AccountType` (the theory) and leaves `AccountKind`/`DocumentKind` open for the calling domain — the mapping should likewise be forms' fixed shape with the domain's own vocabulary on top. | 📋 | Filed 2026-09-28 from the owner's own framing while building the `party-mobilization` agency template. That template's `forms.json` carries the full gap as five `unenforceable` entries with a size on each, and its `catalog.json` carries the other half — where the records would land (see `mwanachama-backend-catalog`'s CAT15, which is the same design from the store's side and should be planned with this row, not after it). Nothing built; no pinning test, since this is a missing capability rather than wrong behaviour. Open question for the design: whether the target module is always `catalog` or whether the mapping names any module (accounting for a contribution form, actor for a registration form) — the party-mobilization template assumes the latter. |
| F13 | 🐞 **SECURITY — an answer is filed under whatever member the request names, and anyone's answers can be read by naming them.** | 🚀 | Found 2026-10-17 by reading the code while designing `mwanachama-frontend-member`; not yet reproduced by a test. `forms.operations.json` binds `submit_answers`' `member_id` from the **body** and `list_answers`' `member_id` from the **path**, and `SubmitAnswers` (`public_impl.go`) files the answer under whichever id it is handed. `actor_id` comes from the session through `{"from": "caller"}` (CLAUDE.md, "Behaviours to preserve"); answers do not. On `mwanachama-wakala-api`'s `/forms` mount, which gates by action under `module:forms` and nothing finer, a caller allowed `forms.answer.submit` can answer as any member, and one allowed `forms.answer.list` can read any member's answers. The retired gateway was not affected: `survey_answer_handlers.go` passed `callerID(r)` itself, so the hole arrived with the declared table. `chapter_id` on the same body has the same shape (an answer counts toward whatever group the request names), though which groups a caller belongs to is a fact only the host has. Fix: bind `member_id` from the caller on `submit_answers`, and split `list_answers` into a member's read of their own answers (caller-bound) and an operator's read under its own action, which is a decision rather than a rename. Blocks `mwanachama-frontend-member`'s MF5, which must not ship before this lands. **Settled by the owner, 2026-10-19** (member app decision 22): `list_answers` becomes the caller's own answers, `member_id` from the session; the operator's read is a new action, `list_all_answers`, held by owner and agent. The owner also handed this row to the member-track routine. |
| F14 | **No read answers "which open forms are waiting for me"; the only list is the operator's register.** | 🚀 | Filed 2026-10-17 for `mwanachama-frontend-member`'s MF4, its Forms tab. `register_forms` (`GET /forms`, action `forms.form.list`) filters by one `chapter_id`, a `status` and a search, and returns how many people answered each form: an operator's view. A member needs the open forms that reached any of their own groups, with who sent each, how many questions, when it closes and whether they have already answered, and nobody else's figures. Open question before building: forms knows which groups a form reached (targets and propagation) but not which groups a caller belongs to. **Settled by the owner, 2026-10-19** (member app decision 19): the mount takes a resolver from the host, the way custody carries `ScopeResolver`, and asks it which groups the caller is in only when a read needs it; groups never come from the request. wakala-api WK96 supplies it. |

_Nothing else open — see [todo_done.md](todo_done.md) for F1–F4 and F6–F12._

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

---

## What the accounting pilot settled (2026-09-28)

`mwanachama-backend-accounting` ran its whole conversion set (W16–W22) on
2026-09-28 and is **done**. It was the cheapest of the three candidates and
went first deliberately. Eight findings change how the rows above should be
read — none of them are guesses, all of them cost time there.

**1. The consumer step is aimed at the wrong repo.** The row below that says
"the gateway builds and tests against the declared shape" was written when
`mwanachama-backend-api-gateway` was the only consumer. The owner's
direction on 2026-09-28 is *"we are using wakala-api on local for now, not
gateway"*. For accounting that meant the real consumer work was wiring
`mwanachama-wakala-api`, which had never imported it at all, and the
gateway became a keep-it-compiling obligation rather than the acceptance.
**Re-read that row before starting it** and decide which repo it means.

**2. Keeping every consumer compiling is a gate, not a step.** §1 of the
consolidated backlog exists because AGD-007 is recorded complete while two
of its five consumers have not built since. Accounting's pass treated
"every consumer still builds" as a precondition on finishing, not as the
last item. Do the same, and note that the gateway is *currently* broken by
the forms conversion (`mwanachamaforms.DefaultTableNames`/`Migrate` are
gone but `cmd/server/stores.go` and `internal/api/http/backend_memory_test.go`
still call them) — whoever owns that should close it.

**3. `specstore` stores an absent string as `''`, not NULL.** So a column
that is *optional but unique when present* cannot be declared `unique`:
every row without a value collides on the one constraint. The workable
shape is a plain indexed column plus a partial unique index created in
`Provision` (`create unique index ... where col <> ''`), with the friendly
refusal done in Go. Accounting needed this twice.

**4. There is no `time.Time` arm in `specstore`.** Timestamps are carried as
RFC 3339 strings. If anything orders on one, use a fixed-width
nanosecond layout (`models.TimeLayout`, as agency does) — plain
`time.RFC3339` is second-precision and will not order rows created in the
same second, and `RFC3339Nano` trims trailing zeros, which breaks
lexicographic ordering. There is no `[]byte` arm either; carry raw bytes as
a hex string.

**5. Pointer carriers now need `nullable` in the blueprint.** The
uncommitted nullable/float work in `mwanachama-backend-shared` (owned by
another session, confirmed staying) made `specstore.New`'s `disagreements`
check strict in *both* directions: a pointer field fails unless the
declared field says `nullable`, and a `nullable` field fails unless the
carrier is a pointer. This refuses carrier/spec pairs that construct fine
today, so it decides whether a store builds at all.

**6. The domain spec file is `<domain>.<module>.json`.** That is what
catalog (`agency.catalog.json`) and agency (`agency.agency.json`) actually
ship, not the `<module>.<domain>.json` several of these row titles guess.
Ship an embedded default plus `SpecFor(instance)` so a consumer needs no
file path at runtime, and a second domain under `spec/examples/` that fills
the same roles with different nouns — that second file is what actually
proves the module learned no vocabulary.

**7. Delete the hand-rolled memory fake.** Catalog has no second
implementation: it runs the real spec store on in-memory SQLite. Accounting
had a `memory.go` and a `postgres.go` implementing one interface, and its
W15 double-reversal bug existed *because* there were two copies of the same
write that could drift. One store, one write path, tested on SQLite.

**8. A bug row whose fix site the conversion rewrites should be folded in,
and its pinning test inverted rather than deleted.** Accounting closed W15
inside its store step and turned
`W15_PostDoesNotRefuseADoubleReversal` into
`TestPostRefusesASecondReversalOfTheSameEntry`. A sibling copy of the same
bug in another repo is *not* closed by that, and stays its own row.

**If the mount is per instance in wakala-api**, note the two halves:
`Shape()` should return `[]Route` (agency's signature, which the
per-instance mount loop consumes) and the routes should be built with
catalog's `Mount{Authorize, Caller}`, so every declared action arrives
gated rather than merely behind `requireCaller`.
