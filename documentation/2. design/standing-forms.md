# Standing forms

Hub Steward board rows HS50 and HS75; Hub Steward decisions 18 and 45, gap 14.

A standing form is one a member fills in unprompted, as often as they like:
the lab's issue form is the first. Every send is its own record. An ordinary
form keeps one send per person, per group, per period, and a resend edits it
([answers.md](answers.md), F30 and F31).

## What is built

- **`form.standing`**, a bool declared in `forms.blueprint.json` and carried
  by `models.Form.Standing`. False by default, so every existing form and
  every row older than the column behaves exactly as before. Like
  `window_advisory`, it is chosen at `Create`; `Update` copies only the
  frame fields it names, so an edit cannot clear it.
- **Each send gets its own key.** `SubmitAnswers` asks `sendPeriodKey` for
  the period it writes under. For an ordinary form that is
  `currentPeriodKey`, unchanged. For a standing form it is a fresh
  `send-<uuid>`. Because `period_key` already sits in the unique indexes of
  both `answer` (`question_group`) and `submission`
  (`form_member_group`) since F87, a distinct key per send is all that keeps
  sends apart: no index, no table and no write path changed.
- F31 still holds, scoped to the send: the answers a send does not carry are
  dropped only from that send's key, which is new, so no earlier send loses
  anything.
- The answered list (F19) lists one row per send, each with its own
  `period_key`, the same way it lists one row per period of a repeating form.

Tests: `hs50_standing_test.go`.

## Why a key and not a new object

The engine already keeps one send per `(form, member, group, period_key)`.
A standing form needs one send per act, so the act names its own period. A
separate counter or a second submission object would have needed a second
uniqueness rule and a second read path for every list that counts sends.

## Who reaches it (F91, member app decision 448)

A service's form is a standing form. Anyone in the service group's audience
fills it from Apply, it is never in their Waiting list, and every application
is its own record.

- **The audience is the host's.** `UseStandingAudience` gives the manager a
  `StandingAudience` function: `(caller, group) -> in or out`. The module
  never learns what the audience is; for a service group the host answers
  from that group's Public, Members or Staff setting (wakala-api WK198). A
  standing form's targets still name its groups; the audience replaces
  belonging (`UseGroups`) as the test of who reaches it, so a caller who
  belongs to the target group but is outside its audience is not reached,
  and one outside it but in its audience is.
- **A mount with `ReachedOnly` and no audience supplied reaches nobody** with
  a standing form, the same fence-closed default comm's `Standing` takes.
  Every miss answers exactly as a missing form.
- **Which group a send records.** `chapter_id` is the first target, in the
  order they were added, whose audience admits the caller; answering `as`
  a named group needs that group to be a target that admits them. A caller
  no target admits is refused (`ErrInvalidAnswer`). With no audience
  supplied at all, an unnarrowed mount keeps the old choice through
  belonging, so the HS50 behaviour is unchanged.
- **Never waiting.** `ListWaiting` skips every standing form, whoever it
  reaches. A standing form is offered by the host where it belongs (Apply),
  never as something owed.
- **Every send is its own record.** Unchanged from HS50: each send gets its
  own `send-<uuid>` period key, so a second application never overwrites
  the first. Decision 208 (one send per person, a resend edits it) does not
  apply to a standing form.
- **Reading a send back by id.** `GET /forms/{form_id}/sends/{send_id}`
  (`GetSend`, action `forms.submission.read_my_send`) answers one send with
  its answers in prompt order. The member is `from: caller`, so only the
  sender reads it; another member's send, an unknown id and a send on
  another form all answer the same 404. The ids come from the answered list,
  whose rows now carry `send_id`. It is not narrowed by reach, like F88's
  reads of one's own answers, so leaving the audience does not hide what was
  sent. It works for an ordinary form too.
- `answers/mine` and `submissions/mine` keep reading the current period, so
  they stay empty for a standing form; a send is read by its id instead.

Tests: `routes/f91_standing_audience_test.go`.

## Sent to nobody (HS50, agency-wide)

The lab's issue form is "a form nobody is sent" (decision 18): it carries no
target at all and is open to every member of its agency.

- **The mount is the agency.** A standing form with no target has no group to
  ask an audience about, so under `ReachedOnly` it is reached by every caller
  who is signed in, and by no one who is not (`callerID` empty answers the
  same 404 as a missing form). Who counts as a member of the agency is the
  host's: it only mounts the routes for them.
- **Fence-closed default unchanged for targeted forms.** The rule above, that
  a mount with `ReachedOnly` and no audience supplied reaches nobody, applies
  to a standing form that names a target. Adding a target to a standing form
  puts it back under its audience.
- **Whichever host functions are wired.** With or without `UseGroups` and
  `UseStandingAudience`, a send to nobody succeeds and is recorded with an
  empty `chapter_id`; naming a group with `as` is refused
  (`ErrInvalidAnswer`) because the form was sent to none.
- **Still never waiting, still its own record per send**, read back by id and
  through the answered list by its sender alone, as above.

Tests: `routes/hs50_agency_wide_test.go`.

## What decision 45 settled (HS75, owner, 2026-10-26)

> A standing form is open at all times; a schedule on it is ignored as it is
> today; and each standing form names where its sends land: a chat thread
> with the agent, or a case on the board. The forms module imports neither
> module; the host wires each landing.

### Open at all times

- **`Create` stores a standing form with `window_advisory` true**, whatever
  the body said. `opens_at` and `closes_at` are still required and still
  validated as before (a closing time in the future, after the opening one),
  because a form is composed with them; they are only shown, never kept.
  `Form.OutsideWindow` is unchanged and reads the stored flag, so a standing
  form never makes `SubmitAnswers` answer `ErrOutsideWindow`, before it opens
  or after it closes, and `ListAnswered` reports it open for as long as its
  status is open. The waiting list never listed a standing form anyway.
- A form that is not standing is exactly as before: it keeps its window
  unless it was composed advisory ([window.md](window.md)).
- **Forms stored before this change.** A standing form written under HS50 or
  F91 was stored with whatever `window_advisory` it was sent. `Provision`
  now sets it true on every standing row that holds it false, in one
  idempotent update, so decision 45 holds for rows already in a database
  and not only for new ones. No other row is touched.
- `Publish` is not changed: it still refuses a form whose `closes_at` has
  already passed (`ErrClosesInPast`), advisory or not. Only sends after
  publishing are exempt from the window.

### A schedule is ignored

A standing form composed with a repeating schedule keeps the schedule, as
today, and ignores it for its sends: every send takes its own `send-<uuid>`
key (`sendPeriodFor`), so repeated sends are never folded into one period and
never refused for being outside one. Composing both is allowed, not refused.
`TestAStandingFormComposedWithAScheduleStillTakesRepeatedSends` pins it.

### Where a send lands

- **`form.landing`**, an enum declared in `forms.blueprint.json` and carried
  by `models.Form.Landing`: `thread` (a chat thread with the agent), `case`
  (a case on the board) or empty (the send lands nowhere beyond its own
  record, as before). Empty is a declared value, which is why a form row
  that predates the column reads back as landing nowhere.
- **Only a standing form may name one.** `Create` refuses a landing on a form
  that is not standing, and a value outside the three, with
  `ErrInvalidForm` (400). The first rule is `models.ValidateLanding`; the
  second is the blueprint's own enum check.
- **Chosen at `Create`, never changed.** Like `standing` and
  `window_advisory`, `Update` copies only the frame fields it names, so an
  edit can neither clear a landing, change it, nor give one to a form that
  has none.
- It is on the form itself, so every shape that carries a form carries it:
  the compose body and the compose tool take `landing` beside `standing`,
  `get_form` returns it, and `Get`, `ListForChapter` and `ListReachable`
  return it to a host. It is left out of the JSON when empty. The register,
  waiting and answered rows never carried `standing` and do not carry it.
- **The module only records the choice.** It imports neither the chat module
  nor the task manager. Opening the thread or the case is the host's.
- Where a deployment's table already holds rows, `Provision` adds the column
  as `text default ''`. An enum that declares the empty value and no default
  is now one of the column kinds `Provision` can add to an existing table,
  beside optional text, bool and nullable number.

### How a host reads the landing and the send id

A host wraps `FormManager` and overrides `SubmitAnswersFor` (the method the
`submit_answers` route calls, and the one every shorter `SubmitAnswers*`
delegates to), as wakala-api's `sendsRecorded` already does for the send
records. No interface changed.

1. Read the form once: `Get(ctx, formID)` returns `Form.Landing` and
   `Form.Standing`. A landing of `thread` or `case` is the host's cue to
   open one; empty means do nothing.
2. Call the wrapped `SubmitAnswersFor`. Every `Answer` it returns carries
   **`send_id`**, the id of the send's own `submission` row, the same id the
   answered list carries as `send_id` and `GET /forms/{form_id}/sends/{send_id}`
   (`GetSend`) reads back. `send_id` is derived at the moment of the send and
   is never stored on an answer (`spec:"-"`); a body that names one is
   overwritten. The other reads (`ListAnswers`, `ListAllAnswers`) leave it
   empty; use `ListSubmissions` or the answered list for those.
3. Each answer also carries the send's `period_key`, `send-<uuid>` on a
   standing form, so `ListSubmissions(ctx, formID, memberID, periodKey)`
   returns that send's submission when only the key is held.
4. **An empty send has no answers to carry the id.** A standing form with no
   required question takes a send with every answer skipped
   (F30), and `SubmitAnswersFor` then returns an empty slice. The host reads
   the member's newest row from `ListSubmissions(ctx, formID, memberID, "")`,
   which sorts oldest first, and takes the last. It can only be mistaken for
   another send if the same member sends the same form twice at once.
5. Ordinary forms carry `send_id` too, on the same answers: a resend returns
   the id of the one send it edited.

Tests: `hs75_landing_test.go` and `routes/hs75_landing_test.go`.

## Still not this module's

- Opening the chat thread or the board case, and what either one carries,
  belong to the host (wakala-api WK198 for the thread, HS51 for the lab's
  case, in wakala-api's `form-case-landing.md`). Nothing in this module
  changed for HS51: the landing, `send_id` and one key per standing send were
  already the seam.
- Whether the thread or case can be answered or closed by the sender is the
  receiving module's rule.
