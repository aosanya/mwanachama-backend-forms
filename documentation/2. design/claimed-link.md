# A public link grants its respondent that one form

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decision
115) and built as F24. Widened the same day by decision 119 (F29): the claim
alone grants the form, with no declaration needed.

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
respondent, the two columns that were carried and never written before, and
records a `claim` row naming the respondent and the form whose link the call
went through.

- **Each link the token is claimed through is recorded.** A device keeps one
  token across every link it opens, so the same member claiming it again
  through another link answers as the first claim did and adds that link's
  form beside the first. One row per respondent and form, unique.

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
a respondent claimed by the caller was claimed through that form's link, **or**
such a respondent holds a declaration on that form. All three still require
the form to be visible to members (open or closed).

So a claim grants exactly the forms whose links it was made through:
`get_form`, `list_questions`, `list_options` and `submit_answers` answer for
those, and every other form is narrowed to the caller's groups as before,
answering as missing. Holding the link's key is the proof the person came by
that link, since the claim address carries it and names the form.

**No declaration is needed** (F29, decision 119). F24 granted a form only
once the token had declared a group on it; that is dropped. A declaration,
when made, is still recorded as before, and one made before F29 still grants
its form, so a claim made then keeps what it opened.

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
