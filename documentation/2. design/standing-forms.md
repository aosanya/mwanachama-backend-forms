# Standing forms

Hub Steward board row HS50; Hub Steward decision 18, gap 14.

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
  both `answer` (`question_member_group`) and `submission`
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

## Not settled, so not built

Decision 18 says what a standing form is; decision 448 settled who reaches
it, the waiting list and reading back. These remain, and none is guessed
here:

1. **The window.** A standing form is still held to `opens_at`..`closes_at`
   unless composed advisory. Should standing imply advisory?
2. **Where a send lands.** Decision 18 sends each one to the lab's
   taskmanager board as a case (HS51–HS54), and decision 448 has each
   application open its own request (WK198). That hand-off is the host's,
   not this module's.
3. **A schedule.** A standing form with a repeating schedule ignores the
   schedule for its sends. Should composing both be refused?
