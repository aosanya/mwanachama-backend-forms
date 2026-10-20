# A public link's questions, signed out

Decided 2026-10-20 by the owner (`mwanachama-frontend-member`'s decision
114) and built as F23.

## The flow it serves

A person who opens a form from a shared link answers every question first
and signs in only at Submit (member app decision 53, MF9, drawn in M01,
"Arrives by link"). Before this row the link opened (`open_public_link`)
but the questions sat behind `list_questions`, which is gated, so the person
saw the form's title and nothing to answer.

## The operation

| Operation | Address | Action | Anonymous |
| --- | --- | --- | --- |
| `public_link_questions` | `GET /public-links/{key}/questions` | `forms.public_link.questions` | yes |

It answers a JSON array, one entry per question, each the question's own
fields plus `options`, its choices in their order (empty for a question with
none). The questions come in the order they are asked (`ordinal`).

## A fourth anonymous action

`routes.AnonymousActions` held three actions, and this file's CLAUDE.md
says adding a fourth is a decision, not a convenience. Decision 114 is that
decision: reading a public link's questions needs no caller, because the
link itself is what the respondent holds. It is the fourth, and the only
read beside `forms.public_link.open`.

Answering stays gated (`forms.answer.submit`); nothing here lets a signed-out
caller write an answer.

## What it returns, and what it does not

- **Only the linked form.** The form is the one the key resolves to; no
  other form's questions, and nothing names one.
- **Only the current version of each question.** A question superseded by
  `version_question` is left out and its successor stands in its place, at
  the same ordinal, with the successor's own options. The rule is the one
  the register's question count already uses: a question is current when no
  other question names it in `supersedes_question_id`.
- **No answers.** Nobody's answers, counts or respondents are read.

## A refusal names no reason

The key is resolved exactly as `open_public_link` resolves it, so an unknown
key, a retired link, a link whose form is missing, a closed form and a form
that is not public all answer `ErrLinkNotFound` (404, `not found`) and
nothing more. Splitting them would let anyone enumerate unpublished forms by
trying keys.

## Narrowing

The read is resolved without `ReachedOnly`'s narrowing, as
`claim_respondent` is (`claimed-link.md`): a public form is aimed at no
group, so narrowing would hide exactly the form the link opens. The link's
key is the grant.

## Hosts

A host that serves anonymous actions only from `routes.AnonymousActions` or
`routes.Split` gets this one with no change. A host that keeps its own list
of anonymous actions (`mwanachama-wakala-api`'s forms mount) has to admit
`forms.public_link.questions` there too.
