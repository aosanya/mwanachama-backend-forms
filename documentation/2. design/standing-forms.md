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

## Not settled, so not built

Decision 18 says what a standing form is; it does not settle these, and none
is guessed here:

1. **Who reaches it.** Gap 14 says a standing form is "a form nobody is sent:
   open to every member of its agency". Today `ReachedOnly` admits only forms
   whose targets reached the caller's groups, so a standing form still needs
   a target to be answered by a resident mount. Should a standing form be
   reached by every member of the agency with no target, and which group does
   a send then record as its `chapter_id`?
2. **The waiting list.** Should a standing form ever be listed as waiting, or
   only offered on the Forms screen as something a member may fill in?
3. **Reading back.** `GET /forms/{form_id}/answers/mine` and `GetSubmission`
   read the current period, which a standing form never has, so they answer
   empty. `ListAllAnswers` and `ListSubmissions` show every send. Should
   "mine" list every send, the latest, or none?
4. **The window.** A standing form is still held to `opens_at`..`closes_at`
   unless composed advisory. Should standing imply advisory?
5. **Where a send lands.** Decision 18 sends each one to the lab's
   taskmanager board as a case (HS51–HS54). That hand-off is not here.
6. **A schedule.** A standing form with a repeating schedule ignores the
   schedule for its sends. Should composing both be refused?
