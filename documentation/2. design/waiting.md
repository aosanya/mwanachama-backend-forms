# Forms waiting for a member

`GET /forms/waiting`, action `forms.form.list_waiting`, manager method
`ListWaiting(ctx, callerID)`. Built for F14, so that
`mwanachama-frontend-member`'s Forms tab has something to read; the only
list before it was the operator's register (`GET /forms`,
`forms.form.list`), which filters by one originating group and counts how
many people answered each form.

## What it answers

The open forms that reached any group the caller is in. One row each,
soonest closing first:

| Field | Where it comes from |
| --- | --- |
| `form_id`, `title` | the form |
| `originator_chapter_id` | the group that sent it |
| `published_by` | who opened it |
| `publisher_name`, `publishing_authority` | who opened it by name, and on whose behalf; see [publisher.md](publisher.md) |
| `published_at`, `opens_at`, `closes_at` | the form's window |
| `questions` | the current prompts, superseded versions not counted, the same count the register uses |
| `reached_via` | which of the caller's own groups it reached, sorted; the member app needs one of these to answer from |
| `answered` | whether the caller has sent it, even with no answers (F30, see [answers.md](answers.md#an-empty-send-is-a-send)) |

It carries **nobody else's figures**: no respondent count, no status
tally, no other member's id. The operator's register stays the only read
that counts people.

"Reached" is a target or a propagation row naming the group. A form
reaches its targets' groups when it is published, so both read the same
set today; reading both keeps a group a form was pushed to.

"Open" is status `open` and, for a form that keeps its window, now
inside that window — the same test `SubmitAnswers` applies. A draft,
a form awaiting a decision, an approved but unpublished one and a closed
one are all left out.

## Where the groups come from

**From the mount, never from the request.** forms knows which groups a
form reached but not which groups a caller belongs to, so the host hands
the manager a resolver:

```go
type Groups func(ctx context.Context, callerID string) ([]string, error)

fm.UseGroups(resolver)
```

The owner's decision of 2026-10-19 (the member app's decision 19): the
mount takes a resolver the way custody carries `ScopeResolver`, and it is
asked only when a read needs it. `mwanachama-wakala-api` supplies it as
WK96.

- The caller is bound with `{"from": "caller"}`, so no query, body or MCP
  argument can name whose list this is, and nothing in the request can
  add a group.
- **A manager given no resolver belongs to no groups**, and answers an
  empty list rather than an error — the same posture as comm's
  `UseBelonging`. A mount that has not wired WK96 yet serves a Forms tab
  with nothing in it, never somebody else's forms.
- A request with no caller answers an empty list without asking the
  resolver.
- A resolver that fails surfaces as the dispatcher's redacted 500; its
  message never reaches the wire.

## What it does not do

- **A target that includes descendants reaches a member only if the
  resolver names the targeted group.** forms holds no hierarchy, so it
  cannot tell that a ward sits under a council. Whether "the groups the
  caller is in" should include the ancestors of the caller's own groups is
  the host's answer, not this module's.
- `opens_at` and `closes_at` are returned, and since F17 also filtered
  on unless the form is advisory: an open form whose kept window has not
  opened, or has passed, is left out, because it cannot be answered. An
  advisory form is listed for as long as it is open. See
  [window.md](window.md).
