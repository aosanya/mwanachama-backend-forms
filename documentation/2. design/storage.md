# Storage

What this module stores, and the handful of decisions a reader of the code
cannot recover from the code. The declaration itself is
`forms.blueprint.json` and `forms.forms.json`; every object and every field
carries its own description there, so this page does not repeat them. The
format is documented once, in
[declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md).

## Timestamps are fixed-width strings, not `time.Time`

`models.TimeLayout` is `2006-01-02T15:04:05.000000000Z07:00`, and every
timestamp this module writes goes through it. The nine-digit fractional part
is not decoration, and neither of the obvious alternatives works:

- **`time.RFC3339`** is second-precision. Two rows written microseconds apart
  in the same second compare equal, and every listing this module sorts by a
  timestamp — forms by `created_at`, targets by `created_at`, propagation by
  `targeted_at`, approvals by `requested_at` — falls through to a tie-break
  that does not track real order.
- **`time.RFC3339Nano`** trims trailing fractional zeros, so two timestamps
  with different digit counts stop comparing correctly **as strings**. Since
  these are text columns, string comparison is what the database does.

A fixed nine digits is both parseable and lexicographically sortable in the
same order as chronologically. The layout was copied verbatim from
`mwanachama-backend-actor/models/time.go`, whose own parity test caught
exactly this.

Storing them as text rather than as a dialect timestamp type is the format's
rule, not this module's: a `timestamp` column is text so that SQLite and
Postgres compare it identically.

## No foreign keys out of this module

`Answer.MemberID` and `Answer.ChapterID` name rows in other modules' tables,
and `Target.ChapterID` and `Propagation.ChapterID` name groups. None of them
is a foreign key, and none is checked. This module depends on no group
directory and no member directory to check against, which is what lets it be
mounted beside any of them — the same choice `mwanachama-backend-actor` made
for `Group.ParentID`.

The consequence is real and deliberate: `AddTarget` will aim a form at a
group id that does not exist. Whoever mounts this module owns that check.

## Three rules that live in SQL

`Provision` applies these as raw statements after `spec.Migrate`, Postgres
only. None is expressible in the declaration, and each has a Go rule beside
it that the SQLite unit tests exercise:

| Rule | Why it is not declared |
| --- | --- |
| A question's options are frozen once its form leaves draft | A trigger. The format has no notion of one table's state gating writes to another |
| An answer fills exactly one value column | A multi-column CHECK. The format declares fields, not relations between them |
| A form has at most one undecided approval | A partial unique index over `decided_at = ''`. The format's only named condition is `not_deleted` |

The trigger refuses `UPDATE` and `DELETE` but not `INSERT`, which is what
lets `AddOption` add a choice to a published form: adding a choice cannot
invalidate an answer already given, while editing or removing one can.

## Vocabulary that is carried but never written

Three things are stored because the schema they were extracted from had
them, and nothing in this module writes them. They are not dead columns to
drop — a database restored from the pre-extraction schema holds values in
them — but nothing here will produce one:

- `PropagationKind`'s `localized` and `pushed`, and `Propagation.LastReminderAt`
- `Respondent.ClaimedByMemberID` and `.ClaimedAt` — the claim-on-signup flow
  they describe is not implemented here, and no public read returns them

Confirmed against the source module's own logic before the extraction: no
method there transitioned a row into those states either.
