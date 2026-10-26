# Whether a form keeps its window

F17, the member app's decision 32 (owner, 2026-10-20): **each form chooses,
when it is composed, whether its `opens_at`/`closes_at` window is enforced.
New forms start enforced.**

## The field

`Form.WindowAdvisory`, column and wire name `window_advisory`, a plain
`bool`.

| `window_advisory` | What decides whether the form takes answers |
| --- | --- |
| `false` (the default) | status `open` **and** now falls inside the window |
| `true` | status `open` alone, as every form behaved before F17 |

It is named for the exception rather than the rule on purpose. The
declared-domain store writes every declared column on every write, so a
`bool` carrier cannot tell "the request left it out" from "the request said
false". Naming the field `enforce_window` would have needed either a
nullable `*bool` (with `nil` meaning enforced on every read, forever) or a
blueprint default of `true` that a write of the zero value overwrites
anyway. With the field named for the exception, the zero value *is* the
default: a body that omits it, a Go literal that leaves it out, and a row
written before the column existed all read as enforced, and no code path
has to remember to fill it in.

**It is set when the form is composed and never after.** `Create` keeps
whatever the body says, except that a standing form is always stored
advisory (HS75, [standing-forms.md](standing-forms.md)); `Update` copies
only the title and the window, so an edit naming `window_advisory` changes
nothing. Publishing, closing and every other transition carry it forward
untouched.

## Where it is enforced

`Form.OutsideWindow(now)` in `models/form.go` is the one test, applied in
two places:

- **`ListWaiting`** (`GET /forms/waiting`) leaves out an open form whose
  kept window has not opened yet or has already closed. An advisory form
  is listed for as long as it is open.
- **`SubmitAnswers`** (`POST /forms/{form_id}/answers`) refuses an answer
  outside a kept window with `ErrOutsideWindow`, **409**, after the status
  check: a form that is not open is still `ErrFormNotOpen`, so the two
  refusals say different things to the member app.

"Before it opens" is now earlier than `opens_at`; "after it closes" is now
later than `closes_at`. Both bounds are inclusive of the instant itself.
An empty `opens_at` sets no lower bound (publishing fills it with the
publish time anyway). Times are compared as instants, not as strings.

Nothing closes a form when its window passes. Status still moves only when
somebody closes it; a kept window only stops it being listed and answered.

- **The public link** (F92, member app decision 469). `ResolveLinkKey`
  answers a form whose kept window has passed `closes_at`
  (`Form.PastClosing`) exactly as a closed one: not found, the same
  `ErrLinkNotFound` a wrong key gets, so the link names no reason. Every
  path through `OpenPublicLink` follows — the link's questions, registering
  and declaring as a respondent, and claiming the token. An advisory form's
  link stays open for as long as the form is open, as its submit does.

## Where it is not enforced

- **The public link before it opens.** `ResolveLinkKey` ignores `opens_at`;
  only the closing date silences a link (below).
- **The operator's register and `get_form`.** They show every form whatever
  its window, so the people who sent it can still see it.

## Forms that predate the field

A form written before F17 has no `window_advisory` column to read.
`Provision` adds it as `boolean default false` — what a fresh migration
creates — so **every existing form reads back enforced**, the same as a new
one. That follows from "defaulting to enforced" with no separate rule for
old rows. Its visible consequence: an existing open form already past its
`closes_at` drops off the waiting list and refuses further answers the
first time the new code runs against it, until somebody closes it or it is
recomposed advisory.

`Provision` could previously add only an optional text column to a table
that already held rows (F18, see [publisher.md](publisher.md)). It now adds
an optional `bool` too, with the same default a fresh table would have; a
missing column of any other type is still an error naming it.
`TestAFormThatPredatesTheFieldKeepsItsWindow` drops the column from a table
holding a form, provisions again, and checks the form reads back enforced.
