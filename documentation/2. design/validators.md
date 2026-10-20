# Validators on a question

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decision 47)
and built as F21, for the member app's MF33.

## What a question declares

A question used to carry one rule, `max_length`. It now carries a set of
declared validators, each a field on the question itself, stored with it and
returned with it by every read (`list_questions`, `add_question`,
`update_question`, `version_question`). The member app reads them off the
question and runs the same checks before Submit; forms runs them again on
every submit, so an app that skips them, or an older one, cannot get past.

| Field | Applies to | Absent means |
| --- | --- | --- |
| `required` | every type | `false`: the question may be left out |
| `min_value`, `max_value` | `number`, `currency` | no bound on that side |
| `min_date`, `max_date` | `date` | no bound on that side |
| `min_selections`, `max_selections` | `multi_select` | no bound on that side |
| `max_length` | `free_text` | no limit (unchanged) |

- **Every bound is inclusive.** `min_value: 0` accepts `0` and refuses
  `-0.01`.
- **A number bound is nullable**, carried by a pointer and declared
  `nullable` in the blueprint, exactly as `max_length` is. A minimum of zero
  is a real bound; an absent minimum is no bound at all, and the two read
  back differently (`0` against an absent key).
- **A date bound is a `YYYY-MM-DD` string**, the same layout an answer's
  `value_date` is stored in, so the check is a plain comparison of the two
  and the app compares what it already holds. A date carries no zero that
  could be confused with "no bound", so empty means absent and the column is
  not nullable.
- **A selection bound is nullable** like a number bound. `min_selections`,
  when given, is at least 1, because a multi-select answer already needs one
  pick; absent leaves that one-pick floor as the only lower bound.

## When a question is refused

`add_question`, `update_question` and `version_question` refuse a question
whose validators cannot mean anything, with `ErrInvalidQuestion` (400):

- a bound on a type it does not apply to (a date bound on a number, a
  selection bound on a single choice);
- a minimum above its maximum, on any of the three pairs;
- a date bound that is not `YYYY-MM-DD`;
- a selection bound below 1.

`update_question` replaces the validators with what the body carries, as it
replaces every other field: a body that leaves one out clears it.
`version_question` takes the successor's validators from its body too; the
predecessor's are not inherited.

Whether `min_selections` exceeds the number of choices offered is not
checked: the choices are replaced wholesale on every update, and a draft may
be written in either order.

## When a submission is refused

`submit_answers` refuses with `ErrInvalidAnswer` (400), and writes nothing,
when any answer falls outside its question's bounds or a required question
is left out. The message names the question by id —
`mwanachamaforms: invalid answer: question <id>: value_number 21 is above
max_value 20` —
so the app can place the refusal beside the field it belongs to. The first
failing question is named; the check stops there.

**Required is held against the submission, not against what is on record.**
`submit_answers` takes every answer at once, so a required question must be
in the same request. Only the version nothing has superseded is held to it:
once a required question is superseded, an answer to the old version does not
satisfy its successor.

## Provisioning an older table

The seven columns are added to a question table that predates them when the
module is provisioned: `required` as `boolean default false`, the number and
selection bounds as nullable numbers, the date bounds as text defaulting to
empty. Every existing question therefore reads back with no validators, which
is what it had before.
