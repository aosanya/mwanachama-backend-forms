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
- **Provisional:** an instant the schedule holds no period for (before its
  start, after its end) is stamped `""`, the same bucket as a form that runs
  once. Whether such a send should instead be refused belongs with the open
  questions below.

## What is not built yet

- The other reads do not scope by period yet: the waiting and answered lists,
  `ListSubmissions` and the register's counts still read every period
  together.
- No read for who in the audience missed a period (decision 334).
- **An existing database keeps its old unique indexes.** The two indexes keep
  their names, so `spec.Migrate`'s create-if-absent does not rebuild them
  with `period_key`; a deployment that already holds these tables needs the
  two indexes dropped once so they are recreated. New databases are correct.

## Open questions

- Decision 334 says a period closes "on its own close date". A schedule
  carries only openings; how a period's own close date is stated (a duration
  after each opening? the form's `closes_at` time of day?) is not decided.
- How a repeating form's `opens_at`/`closes_at` and the F17 window relate to
  the schedule's `start`/`end` is not decided: today both are kept and
  neither constrains the other.
- Whether the opening day in the schedule's zone (rather than the agency's
  zone) is the right period key is with the owner; the key is kept as built.
- Whether a send at an instant with no period is stamped `""` (today) or
  refused is not decided.
