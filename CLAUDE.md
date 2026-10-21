# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-forms

Instruments: a form is composed, cleared, published, answered and closed.
Domain logic AND storage both live in this package, imported directly by
whatever mounts it — no separate service, no gRPC, no proto — the same shape
`mwanachama-backend-catalog` and `mwanachama-backend-agency` already took.
Module path `github.com/aosanya/mwanachama-backend-forms`.

Extracted from `mwanachama-backend-api-gateway`'s `survey` domain on
2026-09-04 and renamed: the gateway's `Survey` is `Form` here, and every
`SurveyID` field is a `FormID`. Nothing else was renamed. Converted onto the
declared-domain format on 2026-09-28 (F6–F12) — see
[documentation/2. design/todo_details/FMD-001.md](documentation/2.%20design/todo_details/FMD-001.md)
for what was decided and why.

## Objects are declared, not written

The tables come from a JSON spec, not from Go structs. The format's reference
is
[declared-domains.md](../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md),
and the route table's is
[dispatcher.md](../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md).

- **`forms.blueprint.json`** — the module's twelve objects, declared once, with
  a description on every object and every field. Reached through
  `Blueprint()`, `LoadSpec(path)`, `ParseSpec(raw)` and `SpecFor(instance)`.
  Load a domain spec through those, never through `spec.Load`, or its roled
  objects arrive with no fields.
- **`forms.forms.json`** — the domain's own declaration: which object fills
  each role, what it is called, and which table it lands in. It names
  `wakala` as its instance by way of example; a mount chooses its own
  through `SpecFor`.
- **`forms.operations.json`** — the API's declaration. Thirty-seven operations,
  each with its method, path, manager method, gating action, status, prose
  and arguments. `routes/` is an adapter over
  `mwanachama-backend-shared/dispatch` and nothing else.

**Adding a field means editing the blueprint and the `models/` type
together.** They are checked against each other in `NewFormManager`, so a
declared column with no field to hold it, or a field with no column to land
in, fails when the manager is built rather than dropping a value on every
write. **Adding an address means editing `forms.operations.json`** — there is
no route builder left to name.

**A table is `<instance>_hashOf(<module>_<mount>_<object>)`** — only the
instance stays readable, e.g. `agy1f2e3d4c_8766f69928a18f58`. The hash is 16
hex characters of SHA-256 over the **raw name** `<module>_<mount>_<object>`,
which `spec.RawNameFor` builds and `spec.Migrate` records in the
`spec_table_names` registry alongside the physical name. Hashing is what keeps
every identifier a constant 28 bytes (51 for an index) against Postgres's
63-byte ceiling, which the readable form had already reached. Assert on
`RawNameFor` in tests, never on a physical name literal, and exclude
`spec.NameRegistryTable` from anything that counts tables. See
[declared-domains.md](../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md).


**A column is found by field name, never by json tag** — `SubmittedBy`
becomes `submitted_by`, `OptionIDs` becomes `option_ids`. The tag is a
presentation choice, and a tag-reading codec would quietly stop storing any
field a response hides.

**Every declared column is written on every write.** A map missing a key
means "leave it alone" to an update, so omitting empty values would make
clearing a field impossible.

**Nine fields are nullable or collection-shaped**, which is why this repo is
the reason `spec` has a `float` type and a `nullable` flag at all:
`Question.MaxLength`, `Question.QuickPicks`, the four number and selection
bounds `Question.MinValue`/`MaxValue`/`MinSelections`/`MaxSelections` (F21,
see [validators.md](documentation/2.%20design/validators.md)),
`Answer.OptionIDs`, `Answer.ValueNumber`, `Answer.ValueBool`. A nullable column must be carried
by a pointer and a pointer must be declared nullable — `specstore` refuses
the mismatch when the manager is built, because otherwise "nobody answered"
and "answered zero" become the same stored row.

## Rules are declared too, where they can be

`validate.go` reads `required` and an enum's `values` off the spec and
applies them on the way in, so a domain that adds a sixth status gets it
enforced with no Go change. `Check(spec, role, value)` is the same
validation without a database.

**What stays in Go is what a spec cannot say**, and each piece lives with
the type it is about: `models.ValidateWindow` (a closing time falls after an
opening one), `models.ValidateAudienceCollection` (a member audience is
never interviewed), `models.ValidateAnswer` (a value fits the shape its own
question asks for and the bounds it declares), `models.ValidateValidators`
(those bounds can be met at all), and the lifecycle transitions themselves.

## Behaviours to preserve

- **A public link's refusal names no reason.** An unknown key, a retired
  link, a missing form and a form that is not open and public are all
  `ErrLinkNotFound`, whose text is `not found` and nothing more. Splitting
  them lets anyone enumerate unpublished forms by trying keys.
- **Answering is not anonymous.** This module cannot tell a public
  respondent from a member, so `forms.answer.submit` is gated like every
  other write. Only four actions are on `routes.AnonymousActions`, and
  adding a fifth is a decision, not a convenience. The fourth,
  `forms.public_link.questions`, was decision 114 (F23).
- **The address outranks the body.** A question or a target posted under one
  form lands on that form whatever the body claims. Declared as `into`, and
  pinned.
- **Server-owned fields are the store's.** `id`, `created_at`, a target's
  `created_at`, an answer's `answered_at`/`chapter_id` on an edit — all
  minted or carried forward by the manager, never read from a request. This
  is where F4 was closed.
- **`actor_id` comes from the mount**, through `from: caller`, so no request
  and no tool can name who did something.
- **An answer's `member_id` comes from the mount too** (F13), on
  `submit_answers` and `list_answers` alike. A body naming another member is
  overwritten, a submission with no caller is refused, and
  `GET /forms/{form_id}/answers/mine` reads only the caller's own. Everyone's
  answers are a separate address and a separate action,
  `forms.answer.list_all`, so a grant to answer never carries a grant to
  read others. See
  [answers.md](documentation/2.%20design/answers.md).
- **A member's list of the forms they answered is theirs alone** (F19).
  `GET /forms/answered` binds the caller with `from: caller` and lists every
  form they have sent to, open or closed, newest sent first; see
  [answered.md](documentation/2.%20design/answered.md).
- **A send is what makes a form answered, even an empty one** (F30).
  `SubmitAnswers` records a `submission` row per form and member in the same
  transaction as the answers, so a send with no answers is still a send; the
  waiting list, the answered list and the register's respondent count all
  read it, and a resend edits it rather than adding one. `Provision` records
  a send for every answer stored before the row existed. See
  [answers.md](documentation/2.%20design/answers.md#an-empty-send-is-a-send).
- **A form is aimed at exactly one group** (F20). A second target is
  `ErrFormHasTarget`, and an answer's `chapter_id` is that one target, never
  the body's.
- **A resident sees only the forms sent to their groups** (F15). A mount
  with `ReachedOnly` set narrows `get_form`, `list_questions`,
  `list_options` and `submit_answers` to published forms that reached one
  of the caller's groups, and a form that did not answers exactly as a
  missing one; see
  [reached.md](documentation/2.%20design/reached.md).
- **A public link grants its respondent that one form** (F24). A signed-in
  caller claims the token they answered under
  (`POST /public-links/{key}/respondents/claim`, the member `from: caller`,
  once per token), and `ReachedOnly` then lets them reach every form whose
  link they claimed it through, declared or not (F29), and no other; see
  [claimed-link.md](documentation/2.%20design/claimed-link.md).
- **A public link's questions are read signed out** (F23).
  `GET /public-links/{key}/questions` answers the linked form's current
  questions with their options, in order, to an anonymous caller, and
  refuses every miss as `ErrLinkNotFound`; see
  [link-questions.md](documentation/2.%20design/link-questions.md).
- **`published_by` comes from the mount too** (F18), and the publisher's
  display name beside it from the host's `Names` resolver, never from the
  body. The publishing authority is body text, one line, at most 120
  characters; see [publisher.md](documentation/2.%20design/publisher.md).
- **`created_by` comes from the mount too** (F22), overwriting a body that
  names another creator, so a form always says who composed it; see
  [publisher.md](documentation/2.%20design/publisher.md#who-composed-it).
- **A form keeps its window unless it was composed advisory** (F17).
  `window_advisory` is false by default, so a new form, an omitted field
  and a row older than the column are all enforced: the waiting list drops
  it outside `opens_at`..`closes_at` and `SubmitAnswers` refuses with
  `ErrOutsideWindow` (409). Only `Create` sets it. See
  [window.md](documentation/2.%20design/window.md).
- **A resubmitted answer edits in place** and keeps the time and group the
  first answer was given from; only `edited_at` moves.
- **A send replaces the member's whole set of answers on that form** (F31).
  An answer the send does not carry is deleted in the same transaction, so
  an answer taken back to Skipped is gone; see
  [answers.md](documentation/2.%20design/answers.md#a-send-replaces-the-whole-set).
- **A failed publish leaves the form unpublished.** The status change, the
  link and the propagation rows are one transaction, so a colliding link key
  is a clean 409 and the form is still publishable.
- **Only the current version of a question may be superseded** — the history
  is a chain, not a tree.

## Consumers

- **`mwanachama-wakala-api`** mounts the declared operations at `/forms`
  (`cmd/server/forms.go`, `internal/api/http/forms_routes.go`), gated
  through `mwanachama-backend-permissions` under the scope `module:forms`.
  Unset `FORMS_DATABASE_URL`/`FORMS_INSTANCE` leaves it unmounted.
- **`mwanachama-backend-api-gateway`** does not use this repo's `routes/` at
  all. It drives `FormManager` through its own
  `internal/store/formsadapter` and serves its own `survey_*_handlers.go`
  addresses. Changing a *manager signature* reaches it; changing an
  *address* does not.

## Conventions

- `go test ./...` (sqlite via `glebarez/sqlite`) is the expected way to
  verify a change here — do not reach for a real Postgres. The Postgres-only
  rules `Provision` applies as raw SQL (the published-option lock, the
  answer has-exactly-one-value CHECK, the one-open-approval index) live in
  `postgres_integration_test.go`, which is `//go:build integration`, gated on
  `POSTGRES_URL`, and **not** part of `go test ./...`.
- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md);
  design decisions live on
  [documentation/2. design/todo.md](documentation/2.%20design/todo.md) as
  `FMD-XXX`.
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md). What the code used to
  say in comments is in
  [documentation/2. design/storage.md](documentation/2.%20design/storage.md)
  — the timestamp layout, the absent foreign keys, and the three SQL rules.
- **There is no `doc.go`.** A package doc is a doc comment, and this repo
  carries none; `documentation/2. design/README.md` is the orientation page
  instead.
- There are no hand-written route builders left to name, so this repo is no
  longer an example of the `"<ModelType>Routes"` convention. Do not
  reintroduce the `mwanachama-backend-actor` `models/` + `gormstore/` +
  route-builder shape here.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
