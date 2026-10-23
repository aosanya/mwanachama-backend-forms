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


## F13 — an answer belongs to the caller — 2026-10-19

The hole filed 2026-10-17: `submit_answers` read `member_id` from the body and
`list_answers` from the path, so a caller allowed either action could answer
as, or read, any member. Settled by the owner as member app decision 22.
`submit_answers` now binds `member_id` with `{"from": "caller"}` (a body naming
another member is overwritten, like `actor_id`), and `SubmitAnswers` with no
caller is refused with `ErrInvalidAnswer`. `list_answers` keeps
`forms.answer.list` but moved to `GET /forms/{form_id}/answers/mine` and reads
only the caller's answers. The operator's read is a new operation,
`list_all_answers` at `GET /forms/{form_id}/answers`, action
`forms.answer.list_all`, manager method `ListAllAnswers`, with an optional
`?member_id=` filter. Recorded in
[answers.md](../2.%20design/answers.md) and CLAUDE.md's "Behaviours to
preserve". On `mwanachama-wakala-api` the per-agency mount already admits only
owners and agents, so "held by owner and agent" needed no grant code there.

Landed as c958e39, adopted by wakala-api e4efe22. Verified:
`go build`/`go vet`/`go test ./...`/`gofmt -l .` clean here;
`routes/f13_answers_are_the_callers_test.go` goes through the real dispatch
table with a real caller and all four tests go red with the fix reverted.
wakala-api's `forms_answers_security_test.go` fails against the unfixed module,
`go test ./...` there is red only on the deliberate `_OpenHole_` tests, and
`make self-test`'s new `instance-form-answers` collection is 13/13. Still open:
`chapter_id` is still read from the body, so an answer counts toward whichever
group the request names; the host's group resolver (F14, WK96) is the fix.

## F14 — a member's list of the forms waiting for them — 2026-10-19

Filed 2026-10-17 for the member app's Forms tab (MF4): the only list was the
operator's register, with response counts. Settled by the owner as member app
decision 19: the host supplies a resolver, asked only when a read needs it,
and groups never come from the request. New operation `list_waiting_forms` at
`GET /forms/waiting`, action `forms.form.list_waiting`, caller-bound, over
`ListWaiting(ctx, callerID)`. It returns the forms with status `open` whose
targets or propagation rows name one of the caller's groups, soonest closing
first, each as a `models.WaitingForm`: id, title, the sending group, who
published it and when, opens and closes, the current question count, which of
the caller's groups it reached, and whether the caller has answered. No
respondent counts and nothing about other members. The seam is
`type Groups func(ctx, callerID) ([]string, error)` set with `UseGroups`,
mirroring comm's `UseBelonging`; no resolver or no caller answers `[]` without
asking it, and a resolver error is the dispatcher's redacted 500. Recorded in
[waiting.md](../2.%20design/waiting.md).

Landed as 44b1f23, with wakala-api's `instance-forms` self-test extended in 3080bc4
(no wakala-api code change; WK96 supplies the resolver). Verified:
`go build`/`go vet`/`go test ./...`/`gofmt -l .` clean here, with
`routes/waiting_test.go` through the real dispatch table (status filter and
"anyone's answer" mutations both go red); wakala-api `go test ./...` red only
on the deliberate `_OpenHole_`/`_OpenBug_` tests and `make self-test`'s
`instance-forms` 16/16. Open for WK96 or the owner: whether a caller's groups
include ancestor groups, so a target with `includes_descendants` reaches child
groups' members; and whether a form past `closes_at` but never closed, or not
yet at `opens_at`, should be listed (today status alone decides).

## F20 — a form targets exactly one group (and F16, settled by it) — 2026-10-20

`AddTarget` refuses a second group on a form that already has one with
`ErrFormHasTarget` (409, declared in `forms.operations.json`); the same group
twice is still `ErrDuplicateTarget`, and removing the target of a draft lets it
be aimed again. `SubmitAnswers` no longer takes a `chapterID`: an answer's
group is the form's one target (`aimedAt`, the oldest if an older form holds
several), a body's `chapter_id` is ignored as F13 ignores `member_id`, and an
untargeted form files answers under no group. **F16 closes with it**: the
member never names a group, so there is nothing to check against their groups.
Commits 8829791 (forms) and 825c190 (wakala-api's `instance-form-one-target`
self-test, 9/9). `go test ./...` green here; wakala-api `go test ./...` red only
on the deliberate `_OpenHole_` tests; `make self-test` 887/898 across 75
collections, the same eight failing collections as master. Left open: the
one-target rule is read-before-write, not a unique index, so two concurrent
`add_target` calls on one draft can both land (the same race
`ErrDuplicateTarget` already had); see `answers.md`.

## F15 — a resident reads and answers only the forms sent to their groups — 2026-10-20

`routes.Mount` gains `ReachedOnly`; the host sets it on the resident's mount
(wakala-api's `instanceFormsMount(role)`), so forms needs no notion of roles.
With it set, `Get` (and so `SubmitAnswers`), `ListQuestions` and
`ListOptions` (judged by the question's own form) hide any form that is not
published or did not reach one of the caller's groups (`Groups`,
`reachedThrough`), answering exactly as a missing form; no resolver, no caller
or no groups reaches nothing. `reach_impl.go`, `reached.md`, and
`routes/f15_resident_reads_only_reached_forms_test.go`, which fails with the
narrowing off. Commits 0740817 (forms) and 7f956c0 (wakala-api's
`resident_forms.go` sets the flag; `resident_forms_reached_test.go`;
`instance-form-resident` 23/23). `go test ./...` green here; wakala-api red
only on the deliberate `_OpenHole_` tests. Open: WK96's resolver counts direct
registrations only, so `includes_descendants` reaches no child group
(decision 31 says so for now).

## F18 — a form names who published it and on whose behalf — 2026-10-20

`Form` gains `publisher_name` and `publishing_authority`, both returned on the
form and on each `GET /forms/waiting` row. `publish_form`'s `published_by` is
now bound `{"from": "caller"}`, and the name beside it comes from the host's
`Names` resolver (`UseNames`, shaped like `UseGroups`); the authority is
publish-body text, trimmed, refused over 120 characters or with a control
character (`ErrInvalidForm`, the form stays a draft). `Create` clears both and
`Update` copies neither. `Publish` gains a fifth argument; `Provision` now adds
a missing optional text column to an existing table. `publisher.md`;
`routes/f18_publisher_and_authority_test.go` and `f18_provision_test.go`,
mutation-checked. Commits 2f4ad10 and a9e2341 (forms), 881a285 and f80b4fc
(wakala-api: `namesIn` reads the caller's `DisplayName` from the agency's
actor directory; self-test `instance-form-publisher` 11/11). `go test ./...`
green here; wakala-api red only on the deliberate `_OpenHole_` tests;
`make self-test` 519/560 across 76 collections, master's seven red
collections plus the known `:memory:` pool flake. Open: until actors carry a
display name (WK100 leaves it empty), `publisher_name` is mostly empty in
practice; `closed_by`, `submitted_by`, `approver_id` and `withdraw`'s
`actor_id` are still body-bound.

## F17 — each form chooses whether its open/close window is enforced — 2026-10-20

`Form` gains `window_advisory`, a plain bool set only by `Create` (`Update`
copies only the title and the window): false, the default, means enforced, so
an omitted field needs no default filled in. `Form.OutsideWindow(now)` is the
one check: `ListWaiting` drops an open, enforced form before `opens_at` or
after `closes_at`, and `SubmitAnswers` refuses such a submit with the new
`ErrOutsideWindow`, 409 in `forms.operations.json`; status is checked first,
so a form that is not open still answers `ErrFormNotOpen`. An advisory form is
decided by status alone, as before. `Provision` now adds a missing optional
bool column too (`addMissingColumns`), so a form created before F17 reads back
enforced. `window.md` (new), `waiting.md`, `publisher.md`, design `README.md`
and a CLAUDE.md behaviour bullet; `f17_window_test.go` and
`routes/f17_window_test.go`. Commit ebf2d62. `go test ./...` green in the main
checkout, gofmt clean; wakala-api builds against it, red only on its
deliberate `_OpenHole_` tests. Open for the owner: an existing open form past
its `closes_at` now leaves the waiting list and refuses answers until someone
closes it, which decision 32 (new forms start enforced) did not say; a one-off
backfill marking old forms advisory was not built. The public link still
checks status and audience only.

## F21 — a form declares validators for its questions — 2026-10-20

`Question` gains `required`, `min_value`/`max_value` (number and currency,
nullable `*float64` so no bound differs from a bound of 0),
`min_date`/`max_date` (`YYYY-MM-DD`, empty is no bound, inclusive) and
`min_selections`/`max_selections` (multi-select, nullable `*int`, at least 1),
in the blueprint and `models.Question`, returned on every question read.
`models.ValidateValidators` refuses a bound on the wrong answer type, a min
above its max, a malformed date or a selection bound below 1, with
`ErrInvalidQuestion`, on add, update (which replaces the validators) and
version (which takes them from its body). `models.ValidateAnswer` checks the
bounds, and `submit_answers` refuses with `ErrInvalidAnswer` naming the
question and writes nothing; a required current-version question must be
answered in the same submission. `Provision`'s `addMissingColumns` gained
`addableColumn` for text, bool and nullable int/float columns on an existing
question table. `validators.md` (new); `validators_test.go`. Commit 138a8b6,
rebased on F17. `go test ./...` green in the main checkout, gofmt clean;
wakala-api builds against it, red only on its deliberate `_OpenHole_` tests.
For the owner: `required` counts only the submission, not answers already on
record, so a resubmit carrying only changed answers is refused; only the first
failing question is named; `min_selections` above the number of choices is not
refused.

## F19 — a member lists every form they answered — 2026-10-20

Owner, member app decision 40. 5f72661 adds `list_answered_forms`,
`GET /forms/answered`, action `forms.form.list_answered`, over
`FormManager.ListAnswered(ctx, callerID)`; the caller is bound by
`from: caller`, so no request names whose list it is. One `AnsweredForm` per
form the caller has an answer on, open or closed, newest sent first, with the
publisher, window, `open` (the same check `SubmitAnswers` makes), the
earliest `answered_at` and the last `edited_at`. A resident (`ReachedOnly`)
mount still lists closed forms. `answered.md`; six route tests through a real
mux, the `member_id` filter mutation-checked. wakala-api 931f0f7 adds the
action to the resident set and covers it in `instance-form-answers` and
`instance-form-resident`. `go test ./...` green here; wakala-api `make
self-test` 645/676 with exactly master's 26 red collections. Not built:
paging and month grouping. Unblocks the member app's MF28.

## F22 — a form says who made it — 2026-10-20

🐞 from journey 08.5 (report `F22-form-created-by-empty.md` in the
developer repo). Landed as 1f17a77 (f6eed63 before rebasing):
`create_form` in `forms.operations.json` binds
`{"from": "caller", "as": "created_by", "into": "CreatedBy"}`, as
`publish_form` binds `published_by`, so a body naming another creator is
overwritten and the MCP tool no longer offers `created_by`. No Go change.
Pinned by `routes/f22_created_by_is_the_caller_test.go`; `publisher.md` gains
"Who composed it". wakala-api d7c6c23 renamed
`TestJourney08_5_TheFormSaysWhoMadeIt` back from `OpenBug` and passes it
unedited; its `instance-form-publisher` self-test composes with an impostor
`created_by` and checks the session wins. Verified: `go build`, `go vet`,
`go test ./...` ok here, `gofmt` clean; wakala-api `go test ./...` fails only
board-pinned reds, `make self-test` 1166/1179 with only the known failures
(the seven master reds plus security-35, security-41).


## F24 — a public link lets its respondent read and answer that one form — 2026-10-20

51760b6. `forms.respondent.claim` (`POST /public-links/{key}/respondents/claim`)
binds the caller as `member_id` and writes `claimed_by_member_id` and
`claimed_at` once (`ErrRespondentClaimed`, 409, for another member). Under
`ReachedOnly`, a form is also reached when a respondent the caller claimed has
a declaration on it, so `get_form`, `list_questions`, `list_options` and
`submit_answers` open for that one form; every other form still answers as a
missing one, and the answer's `chapter_id` stays the form's own target. See
`claimed-link.md`. Pinned by
`routes/f24_link_respondent_reads_that_form_test.go`. Verified: `go build`,
`go vet`, `go test ./...` ok; wakala-api `go build ./...` ok; the member app's
client regenerated (frontend-member 34a78c7), its checks red only on the
pinned OpenBug tests. Left: wakala-api has not admitted residents to
`forms.respondent.claim` yet.

## F25 — a respondent registered without a key gets one minted — 2026-10-20

118d5b7. `UpsertRespondent` minted nothing when no `public_key` was sent and
stored `""`, so the second key-less visitor on any link hit the unique index.
It now mints a 32-byte `crypto/rand` key (64 hex) and returns it on the
respondent and its public view; a supplied key is kept and an existing key
returns its respondent unchanged. Reproduced first, then pinned by
`respondent_key_test.go`. See `claimed-link.md`, "Registering the token".
Verified: `go build`, `go vet`, `gofmt`, `go test ./...` ok; wakala-api
`go build ./...` ok, its reds unchanged. Owner, member app decision 98.

## F23 — a public link's questions, signed out — 2026-10-20

5bce3d4. `forms.public_link.questions` at `GET /public-links/{key}/questions`
(`FormManager.PublicLinkQuestions`, `models.LinkedQuestion`): the linked
form's current questions in `ordinal` order, each with its options in order;
no answers and nothing about any other form. The key resolves through
`OpenPublicLink`, so every refusal is `ErrLinkNotFound`; `ReachedOnly` does not
narrow it. The fourth anonymous action, by member app decision 114. Pinned by
`f23_link_questions_test.go` and `routes/f23_link_questions_signed_out_test.go`.
See `link-questions.md`. Verified: `go build`, `go vet`, `gofmt`,
`go test ./...` ok; wakala-api `go build ./...` ok, its reds unchanged. Left:
wakala-api's `noLinkIssued` (unknown instance) does not answer the new address
(WK128).

## F30 — an empty send counts as answered — 2026-10-20

a75d219, with wakala-api 9b6c52b. Owner, member app decision 123. A new
declared object, `submission` (`form_id`, `member_id`, `chapter_id`,
`submitted_at`, `edited_at`; unique on form and member), written in the same
transaction as `SubmitAnswers`' answers, so an empty send still answers
200 `[]` but is recorded. A resend keeps `submitted_at` and `chapter_id` and
moves `edited_at`; a refused send records nothing. The waiting list's
`answered`, `ListAnswered` and the register's `respondents` read the
submission. `Provision`'s `recordEarlierSends` backfills one submission per
form and member that already had answers. Two additive reads:
`GET /forms/{form_id}/submissions/mine` (`forms.submission.read_mine`, 404
`ErrSubmissionNotFound` before a send) and `GET /forms/{form_id}/submissions`
(`forms.submission.list`, each with its `answers` count). wakala-api admits
residents to `read_mine` (nine actions) with self-test
`instance-form-empty-send` 14/14. See `answers.md` "An empty send is a send".
Verified: `go build`, `go vet`, `gofmt`, `go test ./...` ok; wakala-api adds
no failing test or collection. Left: the member app's generated forms client
needs regenerating; open for the owner whether an empty resend after real
answers should clear them (it leaves them standing).

## F29 — the public link alone grants its form — 2026-10-20

ffdee30, with wakala-api 3016812. Owner, member app decision 119. A new
declared object, `claim` (`respondent_id`, `form_id`, `claimed_at`; unique on
respondent and form): `ClaimRespondent` records the form of the link it was
claimed through, and `ReachedOnly`'s `cameByLink` grants a form that has a
claim or a declaration for the caller's claimed respondent. A declaration is
still recorded and still grants; other forms stay narrowed to the caller's
groups. Pinned by `routes/f29_link_alone_grants_its_form_test.go`
(mutation-checked); wakala-api `TestAResidentWhoDeclaredNothingReadsAndAnswersTheFormTheyClaimedThroughItsLink`
and self-test `instance-form-claim` 25/25. See `claimed-link.md`. Verified:
`go build`, `go vet`, `gofmt`, `go test ./...` ok. Open (owner): a link
answer still files under no group, since public forms carry no target.

## F31 — a send replaces the member's whole set of answers — 2026-10-21

Owner, member app decision 142. fa49c7d (merged as 55b8a7d): `SubmitAnswers`
also deletes, in the same transaction as the answers it writes and the
submission marker, every answer the caller holds on that form for a question
the send leaves out, so an answer taken back to Skipped is gone and an
all-empty send removes them all while the form stays answered (F30). Carried
answers still edit in place (same id, `answered_at`, `chapter_id`; only
`edited_at` moves); a refused send removes nothing; other members and other
forms are untouched. An answer to a superseded question version is removed
too when the send leaves it out. `f31_send_replaces_the_set_test.go`;
`TestSendingAgainNeverDoublesTheSend` now expects 0 answers after an empty
resend. `answers.md` "A send replaces the whole set". Verified: `go build`,
`go vet`, `gofmt`, `go test ./...` ok; wakala-api 8576771 extends self-test
`instance-form-empty-send` 17/17, no new red. Unblocks member app MF70.

## F32 — publishing a closed form's report — 2026-10-21

558bc6d. A declared `result_report` object (form, group, version, git path and
commit, publisher, time; unique on form and version). `POST
/forms/{form_id}/results` (`forms.results.publish`) refuses a form that is
not closed (409) or has no target (409), takes `published_by` from the
caller, aims the report at the form's one target, adds the next version and
fills `results_published_at` in one transaction. `GET
/forms/{form_id}/results/latest` (`forms.results.read`) answers the newest
version to a caller the host's `Groups` puts in that group, and every miss as
a missing form. `results.md`. Who may publish is the host's gate (WK155).
Verified: `go build`, `go vet`, `gofmt`, `go test ./...` ok; wakala-api builds
against it.

## F33 — approval is each agency's choice — 2026-10-21

a24bcc9 (merged as 1a13d23). `FormManager.UseApprovalRequired(bool)`, off by
default and unchanged when off. On: publishing an uncleared draft is
`ErrNotApproved` (409), and the form's author (`created_by` or submitter)
approving is `ErrSelfApproval` (403). `approve_form` and `refuse_form`
take `approver_id` from the caller. `approval.md`. Storing the setting per
agency and calling it is wakala-api WK156. Open with the owner: when off, may
an author still submit and clear their own form? Verified: `go build`,
`go vet`, `gofmt`, `go test ./...` ok; wakala-api builds and its forms tests pass.

## F39 — a report reaches the groups chosen at Publish, and their inner groups — 2026-10-22

57a972c (merged as e0c2fec). `publish_results` takes optional `chapter_ids`,
any group of the agency; omitted means every target, an emptied list is
`ErrNoTarget`. `ResultReport.ChapterIDs` stored; a resident reads the latest
report when the host's `Groups` names a chosen group (inner groups through the
host, WK167); reports from before keep the every-target rule. `results.md`.
Left for wakala-api WK155: checking each chosen id is a group of the agency.
Verified: `go build`, `go vet`, `gofmt`, `go test ./...` ok; wakala-api builds
and its forms tests pass.

## F40 — one answer per member per target group: answering as — 2026-10-22

1fa7346 (merged as 6cf31d1). `POST /forms/{form_id}/answers?as=<group>` calls
`SubmitAnswersAs`; the group must be a target and in the host's `Groups`
(`ErrInvalidAnswer` otherwise); no group keeps F34's first-target rule. Unique
indexes per (form, member, group) and (question, member, group); a resend as
one group never touches another; waiting forms carry `answered_as`, the
answered list is one row per form and group, `ListSubmissions` counts per
group. `Provision` drops the retired one-per-person indexes. `answers.md`.
Open with the owner: should `GET /forms/{id}/submissions/mine` and
`/answers/mine` take `?as=`? Verified: `go build`, `go vet`, `gofmt`,
`go test ./...` ok; wakala-api builds and its forms tests pass.

## F88 — a member's own answers outside their groups' reach — 2026-10-22

🐞 Fixed. F87 slice 2 read the form through `m.Get` for its period, which a
`ReachedOnly` mount narrows, so `answers/mine` and `answers/mine/previous`
answered 404 `ListAnswers: mwanachamaforms: form not found` once the form no
longer reached the caller. Both now read the form unnarrowed (as
`results_impl.go` and `claim_impl.go` do); the answers stay bound to the
caller (F13), and a refusal names no Go method. `reached.md` widened.
9fe8802 (merged as 3ab4860); test
`routes/f88_own_answers_outside_reach_test.go` through the real route table.
Verified: `go build`, `go vet`, `gofmt -l`, `go test ./...` ok.

## F35 — a public link's ward is a real ward — 2026-10-23

The forms half of decision 236's public-link check. `UseDeclarable` names the
groups a public form offers and refuses any other declared group (400,
403b22e); `ListDeclarations`/`CountDeclarations` on the manager; the declared
operations `GET /forms/{form_id}/declarations` (`forms.declaration.list`) and
`/declarations/count` (`forms.declaration.count`), both gated (9073556,
merged as d45bd2f); `declarable.md`. What remains is wakala-api's: switching
the check on and granting the two actions, which is WK167 there, with the
owner's questions on which groups a link offers and who reads the counts.
Verified 2026-10-23: `go build`, `go vet`, `go test ./...` ok.
