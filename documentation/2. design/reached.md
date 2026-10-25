# A resident reads only the forms sent to their groups

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decision 28)
and built as F15.

## The hole this closes

A mount that lets a resident read and answer forms
(`mwanachama-wakala-api`'s per-agency mount, WK97) gates by action and
nothing finer. `get_form`, `list_questions`, `list_options` and
`submit_answers` all take a form or question id from the address, so a
resident holding any form's id could read it, its prompts and choices, and
answer it while it was open, whether or not it was ever sent to them.

## What the mount says now

`routes.Mount` gained `ReachedOnly`. The host sets it on the mount it
builds for a caller it treats as a resident; forms has no notion of roles,
so which callers are narrowed is the host's answer, the same way which
actions they may reach is its authorizer's. `wakala-api` sets it whenever
it picks `authorizeResidentForms`.

With it set, every route carries the caller (from `Mount.Caller`) on the
context through `forms.OnlyReached(ctx, callerID)`, and the manager narrows:

| Call | Reached | Not reached |
| --- | --- | --- |
| `Get` (`get_form`) | the form | `ErrFormNotFound` |
| `SubmitAnswers` | as before | `ErrFormNotFound`, nothing written |
| `ListQuestions` | the prompts | `[]` |
| `ListOptions` | the choices | `[]` |

**Not reached answers exactly as missing.** An unknown form id is a 404 to
`get_form` and `submit_answers` and an empty list to `list_questions`; an
unknown question id is an empty list to `list_options`. A form that exists
but did not reach the caller gives the same status and the same body, so a
resident cannot tell an unsent form from an invented id. `list_options`
judges the question's own form, not the form named in its address, whose
segment the call ignores.

## What "reached" means

The same test the waiting list (`waiting.md`) uses, through the same
resolver and `reachedThrough`: a target or a propagation row names one of
the groups `Groups` returns for the caller. On top of that the form must be
visible to members (`VisibleToMembers`: open or closed). A draft, a form
awaiting a decision and an approved but unpublished one have not been sent
yet, even when a target already names the caller's group. A closed form
that reached them stays readable, so their answered forms still open; the
waiting list alone shows only open ones.

No resolver, no caller, or a caller in no group: nothing reaches them
through a group. The narrowing fails closed.

A form also reaches a caller who answered it through its public link and
then claimed the token they answered under (F24): a respondent claimed by
the caller was claimed through that form's link, or holds a declaration on
it. Since F29 the claim alone is enough; no declaration is needed. See
[claimed-link.md](claimed-link.md).

## Not changed

A mount without `ReachedOnly` (the operator's) is not narrowed at all.
`list_answers` and `list_previous_answers` (own answers) are caller-bound
already and are not narrowed, including the form they read for its schedule
(F88): a member who left the group a form came through still reads what
they sent, and a refusal names no Go method.

## A sent form stays readable after leaving its group

A form the caller has sent (a `submission` row of theirs on it) stays
readable once it no longer reaches them (F93, member app decision 470):
`get_form`, `list_questions`, `list_options` and `submissions/mine` answer
as before. `submit_answers` still asks whether the form reaches the caller
and answers a form that does not exactly as a missing one, so a sent form
is history, not something to answer again. A form never sent stays hidden.
`chapter_id` on a submission is still the body's (F16).
