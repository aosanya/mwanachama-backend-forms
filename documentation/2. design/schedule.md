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

## What is not built yet

- `SubmitAnswers` does not yet stamp the current period, and the reads
  (`/answers/mine`, the waiting and answered lists, the register's counts, F31's
  replace-the-set delete) do not yet scope by period. Until they do, a
  repeating form still holds one set of answers per member and group.
- No address yet for a member's previous period's answers, nor for who in the
  audience missed a period (decision 334).
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
