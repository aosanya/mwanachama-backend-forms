# A public link grants its respondent that one form

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decision
115) and built as F24.

## The flow it serves

A person opens a form from a shared link without an account, registers the
device as a respondent (`register_respondent`), says which group they are in
(`declare_chapter`), answers, and only at Submit signs in (member app MF9,
decision 53). Signed in, they are a resident narrowed by `ReachedOnly`
(`reached.md`), and a public form carries no target, so without this row the
form they just answered would answer them as missing.

## Registering the token

`register_respondent` takes the `public_key` the device already holds and
answers with the respondent it names, unchanged, if one exists. A body with
no `public_key` is a first visit: the store mints one, 32 bytes from
`crypto/rand` as 64 hex characters, and returns it in the response, and the
device keeps it from then on. Before F25 an absent key was stored as `""`,
and since `public_key` is unique the second key-less visitor on any link was
refused; `respondent_key_test.go` pins both halves.

## Claiming the token

| Operation | Address | Action | Anonymous |
| --- | --- | --- | --- |
| `claim_respondent` | `POST /public-links/{key}/respondents/claim` | `forms.respondent.claim` | no |

The body names the `public_key` the device answered under; the member is the
caller, bound `{"from": "caller"}`, so a body naming a `member_id` is
ignored. The call writes `claimed_by_member_id` and `claimed_at` on that
respondent, the two columns that were carried and never written before.

- **A token is claimed once.** Claiming it again as the same member answers
  as the first claim did. Another member is refused with
  `ErrRespondentClaimed` (409). The write is conditional on the column still
  being empty, so two claims racing for one token cannot both land.
- **The link must still open.** An unknown, retired or closed link is
  `ErrLinkNotFound`, worded and statused like every other refusal of a link.
  The link is resolved without the caller's narrowing, since the form is
  exactly the one their groups do not reach.
- **No caller, no claim.** A mount with no caller, or an empty `public_key`,
  or a token nobody holds, is `ErrInvalidReference` (400).
- **The answer is the token's public view.** `claimed_by_member_id` is never
  returned, here or by any public read.

## What the claim grants

`ReachedOnly`'s check gains a second way in. A form is reached by a caller
when, as before, a target or propagation row names one of their groups, **or**
a respondent claimed by the caller holds a declaration on that form. Both
still require the form to be visible to members (open or closed).

So a claim grants exactly the forms that token named a group for through
their links: `get_form`, `list_questions`, `list_options` and
`submit_answers` answer for those, and every other form is narrowed to the
caller's groups as before, answering as missing. The declaration is the
proof the person came by that link, since declaring is the one write that
needs the link's key and names the form.

## What the answer counts toward

Unchanged by the claim. `submit_answers` files the answer under the caller
and the form's one target (`answers.md`); the group they declared stays their
declaration and is never the answer's `chapter_id`. A public form is aimed
nowhere, so its answers carry no group.

## Not changed

- The waiting list (`waiting.md`) still lists only forms reached through the
  caller's groups. The answered list (`answered.md`) shows a link-answered
  form because it reads answers.
- Claiming is gated; it is not one of the anonymous actions.
