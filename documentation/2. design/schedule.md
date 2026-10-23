# A form's schedule and its periods (F87, first slice)

Member app decisions 332–337 (2026-10-22): a form may repeat. Staff give it a
schedule; each opening starts a period; a period closes when the next opens;
every period's answers are kept apart under the same form.

## What is built

- **`form.schedule`** is a declared `json` field carried by
  `models.FormSchedule`: `start` (RFC 3339), optional `end` (RFC 3339), `zone`
  (an IANA time zone, the agency's), and exactly one of `rule` (an iCalendar
  RRULE, decision 335) or `dates` (`YYYY-MM-DD` list, e.g. school terms). A
  zero value means the form runs once; `Repeats()` tells the two apart.
- **Validation** goes through `mwanachama-backend-shared/recur` (S66):
  `FormSchedule.Recur()` builds a `recur.Schedule` and validates it. `Create`
  and `Update` refuse a schedule that does not validate with
  `ErrScheduleInvalid` (400). `Update` now carries the schedule like the other
  draft fields.
- **A period's key is the day it opened, in the schedule's zone**
  (`2026-10-12`). `PeriodAt(at)` answers the period holding an instant, with
  `OpensAt` and `ClosesAt` (the next opening, or the schedule's end; zero when
  the schedule is open-ended and this is its last known opening).
  `PreviousPeriod(at)` answers the one before it, which is what Copy last
  answers (decision 333) reads. Before the first opening or after the end
  there is no period: `ErrNoPeriod`.
- **`submission.period_key` and `answer.period_key`** are declared string
  columns, and both unique indexes (`form_member_group`,
  `question_member_group`) now include it, so one member answers once per
  group **per period**. Every write today stores `""`, which is the period of a
  form that runs once, so a non-repeating form behaves exactly as before.

## A send is stamped with its period

- `SubmitAnswers` stamps `period_key` on every answer and on the submission
  with the period the form's schedule holds at the moment of sending
  (`PeriodAt(now)`). A form that runs once stamps `""`. A `period_key` in the
  body is overwritten, like every other server-owned field.
- A send edits, and F31's replace-the-set delete removes, only the rows of
  the **same** period, so last period's answers survive this period's send.
- `GET /forms/{form_id}/answers/mine` (`ListAnswers`) reads only the current
  period's answers.
- `GET /forms/{form_id}/answers/mine/previous` (`ListPreviousAnswers`, action
  `forms.answer.list_previous`) reads the caller's answers from the period
  before the current one, for Copy last answers (decision 333). It is empty
  for a form that runs once and before a second period has opened. The member
  is `from: caller`. **A host must grant the new action** before a member can
  reach it.
- `GET /forms/{form_id}/submissions/mine` (`GetSubmission`) answers the
  caller's send in the **current** period; a send from an earlier period is
  not this period's, so it answers not found until the member sends again.
- A submission's answer count is counted within its own period, so
  `ListSubmissions`, which still returns every period's sends, never adds last
  period's answers to this period's send.
- `GET /forms/{form_id}/answers` (`ListAllAnswers`, the operator's read)
  takes an optional `period_key` query argument narrowing it to one period;
  absent, it still reads every period together.
- `GET /forms/{form_id}/submissions` (`ListSubmissions`) takes the same
  optional `period_key` query argument; absent, it reads every period's sends.
  `GetSubmission` (`submissions/mine`) reads through it with the current
  period's key.
- `GET /forms/waiting` (`ListWaiting`, F19/F30) reads the caller's sends in
  the **current** period only, so a member who sent in an earlier period is
  waiting again once the next period opens; `answered_as`/`unanswered_as`
  are this period's.
- `GET /forms/answered` (`ListAnswered`) keeps one row per send across every
  period, each carrying its `period_key` (absent for a form that runs once);
  within one form and group the newest period comes first.
- The register's `respondents`, per row and in total, count sends in each
  form's **current** period; a form that runs once counts its `""` period,
  which is every send it has. An optional `period_key` query argument
  (`RegisterQuery.PeriodKey`) counts every matched form's sends in that named
  period instead, so an earlier period can be read back. The forms listed and
  the totals by status do not change with it; a form that has no such period
  (one that runs once, or a schedule on another rhythm) is still listed and
  counts 0. The key is matched as written, the same way `list_all` and
  `submissions` match theirs.
## What is not built yet

- **An existing database keeps its old unique indexes.** The two indexes keep
  their names, so `spec.Migrate`'s create-if-absent does not rebuild them
  with `period_key`; a deployment that already holds these tables needs the
  two indexes dropped once so they are recreated. New databases are correct.

## A send names its period (member app decisions 361, 370)

- `POST /forms/{form_id}/answers` takes an optional `period_key` query
  argument (`SubmitAnswersFor`): the period of a repeating form the send was
  opened for. When it is not the period open now, the send is refused with
  `ErrPeriodClosed` (409), even when it closed only because the next period
  opened; nothing is written. Absent, the send lands in the period open now.
- A send to a repeating form while no period is open (before the schedule's
  start, after its end, or between a period's `open_days` close and the next
  opening) is refused with `ErrOutsideWindow` (409), the refusal a form past
  its window already gets. Nothing is stamped `""` on a repeating form any
  more.
- A form that runs once and a standing form ignore `period_key`.

## Open questions

- Whether a send to a repeating form that names no period should be refused
  rather than landing in the period open now. Today it lands, so the member
  app keeps working until it sends the key it opened the form for.
- Whether a period before a schedule's first opening should be refused by
  who-missed. Today a key is matched as written against the period open now,
  so only a key at or after the current period is refused.

## The schedule is shared's wire shape (S67, member app decision 367)

- `models.FormSchedule` is a defined type over
  `mwanachama-backend-shared/recur.Wire`, so its fields and JSON names are
  shared's: `start`, `end`, `zone`, `rule` or `dates`, and `open_days`. The
  wire is unchanged apart from `open_days`, which is new and optional.
  `Recur()` is `recur.Wire.Schedule()`; forms keeps no copy of the reader.
  taskmanager (W72) and comm (CM102) read the same type.
- **A list of dates needs an end** (decision 362). shared's reader refuses
  `dates` without `end`, so `Create` and `Update` now answer
  `ErrScheduleInvalid` (400) for one; a rule may still run open-ended.
- **`open_days` is how many days each period stays open** (decision 359).
  A period closes `open_days` days after it opens, on the same wall-clock
  time in the schedule's zone, or when the next opens, whichever is first;
  `0` or absent keeps the old rule (the next opening). The closing time is
  `recur.Schedule.Period`'s; `PeriodAt` answers `ErrNoPeriod` between a
  period's close and the next opening, so a send there is refused like any
  send between periods. `PreviousPeriod` answers the period that opened
  before the current one even when it closed early, so Copy last answers
  still finds it.

## Who missed a period (decision 334)

- `GET /forms/{form_id}/periods/{period_key}/missed` (`ListMissed`, action
  `forms.submission.list_missed`) answers everyone in a repeating form's
  audience with no send in one period that has closed, each with the target
  group that reached them (`models.MissedMember`, `member_id`, `chapter_id`),
  ordered by member. It is an operator's read, gated like `list_all`; it is
  not on `AnonymousActions`. **A host must grant the action.**
- **The audience is the host's.** `UseMembers(Members)` gives the manager a
  `Members` function, `(ctx, groupID, includesDescendants) ([]string, error)`,
  called once per target with that target's own `includes_descendants`, so
  the host decides what a group's sub-groups are. A member reached through
  several targets is listed once, under the first target that reached them.
- A member counts as having sent when they have a submission in that period
  through any group, so a send is never asked for twice.
- **A manager given no `Members` answers an empty list**, the same posture as
  `Groups`: with nobody listed, nobody is known to have missed.
- A period that has not closed is refused with `ErrPeriodNotClosed` (409): the
  period open now, any later key, and an empty key. After the schedule's end,
  or between a period's `open_days` close and the next opening, every earlier
  period counts as closed. A form that runs once is refused with
  `ErrNotRepeating` (409): it has no periods to miss.
