# Every form a member answered

`GET /forms/answered`, action `forms.form.list_answered`, manager method
`ListAnswered(ctx, callerID)`. Built for F19, the owner's decision of
2026-10-20 (`mwanachama-frontend-member`'s decision 40), so that the member
app's "Every form you answered" screen (MF28) has something to read.
`list_answers` reads one form's answers; `list_waiting` reads only open
forms; nothing listed the forms a member had answered once they closed.

## What it answers

One row for each form the caller has sent to — with answers or, since F30,
with none (see [answers.md](answers.md#an-empty-send-is-a-send)) — open or
closed, **newest sent first** — ordered by `published_at`, descending, ties
broken by form id:

| Field | Where it comes from |
| --- | --- |
| `form_id`, `title` | the form |
| `originator_chapter_id` | the group that sent it |
| `published_by`, `publisher_name`, `publishing_authority` | who opened it, by name and on whose behalf; see [publisher.md](publisher.md) |
| `published_at` | when it was sent |
| `opens_at`, `closes_at`, `closed_at` | its window, and when it was closed if it was |
| `status` | where it has got to: `open` or `closed` |
| `open` | whether it can still be answered: status `open` and, for a form that keeps its window, now inside it — the same test `SubmitAnswers` and `list_waiting` apply; see [window.md](window.md) |
| `answered_at` | when the caller first sent it |
| `edited_at` | when the caller last sent it again, empty if they never did |

It carries **nobody else's figures**: no respondent count and no other
member's id. Opening a row reads the answers themselves through
`GET /forms/{form_id}/answers/mine`.

## Whose list it is

- **The caller is a session fact**, bound with `{"from": "caller"}` exactly as
  `list_answers`' `member_id` is (see [answers.md](answers.md)). No query,
  body or MCP argument can name whose answers are listed, and the argument
  is absent from the MCP tool's schema.
- **A request with no caller answers an empty list.** No send can be filed
  under the empty id, since `SubmitAnswers` refuses one.
- The read needs no group resolver. A form is listed because the caller
  answered it, not because it reaches them now, so a `ReachedOnly` mount
  (see [reached.md](reached.md)) lists it too, closed or not.

## What it does not do

- It is not paged. A member answers a handful of forms a month; a page
  comes when a real list is long enough to need one.
- It does not group by month. The member app draws the months from
  `published_at`.
- A form deleted after it was answered is left out. Only an unpublished form
  can be deleted, and an unpublished form cannot be answered, so this does
  not arise today.
