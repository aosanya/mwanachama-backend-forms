# Whose an answer is

Decided 2026-10-19 by the owner (`mwanachama-frontend-member`'s decision 22)
and built as F13.

## The hole this closes

The declared table bound `submit_answers`' `member_id` from the request body
and `list_answers`' from the path. A mount that gates by action and nothing
finer — `mwanachama-wakala-api` under `module:forms` — therefore let anyone
allowed `forms.answer.submit` answer as any member, and anyone allowed
`forms.answer.list` read any member's answers by naming them. The retired
gateway's own handlers had always passed the session's caller, so the hole
arrived with the declared table rather than with the domain.

## What the table says now

| Operation | Address | Action | `member_id` |
| --- | --- | --- | --- |
| `submit_answers` | `POST /forms/{form_id}/answers` | `forms.answer.submit` | the caller |
| `list_answers` | `GET /forms/{form_id}/answers/mine` | `forms.answer.list` | the caller |
| `list_all_answers` | `GET /forms/{form_id}/answers` | `forms.answer.list_all` | optional `?member_id=` filter |

- **The caller is a session fact**, bound with `{"from": "caller"}` exactly as
  `actor_id` is. A body that still names a `member_id` is not refused; the
  key is ignored and the caller is what lands. Being a caller-bound argument,
  it is also absent from the MCP tool's schema.
- **A submission with no caller is refused** with `ErrInvalidAnswer` (400).
  A mount that supplies no caller would otherwise file every answer under
  the empty id, and every such mount's members would share one set of
  answers.
- **Reading everyone's answers is a separate action**, not a wider form of
  the member's read. A grant to answer, or to read one's own answers, never
  carries a grant to read others'. Who holds `forms.answer.list_all` is the
  mount's answer; on `mwanachama-wakala-api` it is the owner and the agent.
- `list_all_answers` returns answers grouped by member, each member's in the
  order the prompts were asked. The `member_id` filter is the operator's
  former read of one person's answers, kept as a narrowing of the new one.

## Which group an answer counts toward

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decision 46)
and built as F20, which also settles F16.

- **A form is aimed at exactly one group.** `add_target` refuses a second
  target on a form with `ErrFormHasTarget` (409); the same group twice is
  still `ErrDuplicateTarget`. Removing the one target, while the form is a
  draft, frees it to be aimed elsewhere. `includes_descendants` still decides
  whether the groups beneath that one are reached.
- **An answer's `chapter_id` is the form's target, never the request's.**
  `submit_answers` no longer reads `chapter_id` at all: the manager looks up
  the form's target and files every answer under it. A body that still names
  a `chapter_id`, at the top level or on an answer, is not refused; the key
  is ignored, the same treatment `member_id` gets. Being no longer an
  argument, it is also gone from the MCP tool's schema. The member never
  chooses which group their answer counts toward, so nothing needs checking
  against the caller's groups.
- **A form aimed nowhere files answers under no group.** A public form
  cannot carry a target, and a member form may be published without one;
  either way `chapter_id` is empty.
- **A resubmission keeps the group the first answer was filed under**, as it
  keeps its `answered_at`.
- A form stored before this rule may still hold several targets. Its
  answers count toward the oldest one, the first `list_targets` returns.

The rule is checked by reading the form's targets before writing, not by a
unique index, so two aims racing on one draft could both land; the same is
already true of `ErrDuplicateTarget`.

## An empty send is a send

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decisions
122 and 123) and built as F30.

Before this, `SubmitAnswers` with no answers returned 200 `[]` and wrote
nothing, so a member who opened a form, chose to answer nothing and sent it
saw the form stay under Waiting, and the operator could not tell them from
somebody who never opened it.

- **Every accepted send records a `submission`**: one row per form and
  member (`form_id`, `member_id`, `chapter_id`, `submitted_at`,
  `edited_at`), unique on the pair. It is written in the same transaction
  as the send's answers, so a send either lands whole, marker included, or
  not at all. A send that is refused — a required question unanswered, a
  form not open, outside its window — records nothing.
- **Sending again edits, never doubles.** A resend while the form is open
  keeps the row, its `submitted_at` and its `chapter_id`, and moves
  `edited_at`; each answer it carries edits that member's answer to the
  same question in place, as before. Empty-then-answers and
  answers-then-empty both leave one row. An empty resend leaves the answers
  already given standing: a send writes what it carries and removes
  nothing.
- **"Answered" now means "sent".** The waiting list's `answered`, the
  answered list (`GET /forms/answered`) and the register's `respondents`
  read the submission, not the answers. The answered list's `answered_at` is
  the first send and its `edited_at` the last resend.
- **The member's read of their own answers is unchanged.**
  `GET /forms/{form_id}/answers/mine` still answers `[]` both for a member
  who sent nothing and for one who never sent, because changing either would
  break a client that reads it to prefill the form. Whether the caller has
  sent is a new address beside it:

| Operation | Address | Action | Answers |
| --- | --- | --- | --- |
| `get_my_submission` | `GET /forms/{form_id}/submissions/mine` | `forms.submission.read_mine` | the caller's send, or 404 `ErrSubmissionNotFound` if they never sent |
| `list_submissions` | `GET /forms/{form_id}/submissions` | `forms.submission.list` | everyone's sends, first sender first, optionally `?member_id=` |

- Each send carries `answers`, how many answers the member has on record
  for that form, so the operator's read shows a member who responded with
  nothing as a send with `answers: 0`. `list_all_answers` is unchanged and
  still lists only answers given.
- The member is bound from the caller exactly as on the answer addresses;
  a request with no caller reads no send. The two reads are new actions
  because the dispatcher holds one operation to one action; they mirror
  `forms.answer.list` and `forms.answer.list_all`, and a mount grants them
  to the same people.
- **Answers stored before the marker** count as sends once provisioned:
  `Provision` inserts a submission for every form and member that holds an
  answer and no submission, taking the earliest `answered_at`, the latest
  `edited_at` and the answer's `chapter_id`. It is one statement, idempotent,
  on SQLite and Postgres alike; the id it gives such a row is
  `<form_id>:<member_id>`, which is as opaque to callers as any other.
