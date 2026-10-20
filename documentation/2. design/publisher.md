# Who published a form, and on whose behalf

F18, the member app's decision 39 (owner, 2026-10-20). A resident reading a
form wants to know who is asking. The actor directory is the host's, not
this module's, and a resident cannot read it, so the answer is stored on the
form as text when it is published.

## The two fields

| Field | Where it comes from |
| --- | --- |
| `publisher_name` | the host's directory, asked once at publish time for the caller's display name |
| `publishing_authority` | the publish body: the office or group the form is sent on behalf of, e.g. "Treasurer, Kilimani Ward" |

Both are on the form (`GET /forms/{form_id}`, and every read that returns a
form) and on each row of `GET /forms/waiting`. The member app shows the
authority, or the publisher's name when no authority was given.

They are a **copy taken when the form opened**. Renaming the person later
does not rename the forms they already sent, and publishing twice changes
nothing, the same as before.

## Why the name comes from the mount

`published_by` is bound with `{"from": "caller"}` since F18, as `actor_id`
and an answer's `member_id` already were. A body naming another publisher is
overwritten, so the id and the name stored beside it are always the person
who actually opened the form; a display name read from the body would be
whatever the request claimed.

forms knows ids, not names, so the host hands the manager a resolver, the
same shape as `UseGroups`:

```go
type Names func(ctx context.Context, callerID string) (string, error)

fm.UseNames(resolver)
```

- **A manager given no resolver stores no name** and still publishes. The
  app falls back to the authority, and to nothing when there is neither.
- A resolver's error fails the publish, and the form stays unpublished.
- The name is put on one line (control characters become spaces), trimmed,
  and cut to 120 characters (`models.MaxPublisherName`). It is the
  directory's text, not the caller's, so it is shortened rather than
  refused.

## Why the authority comes from the body

There is no declared object for an office yet, so the authority is what the
person publishing writes. It is trimmed and refused with `ErrInvalidForm`
(400) when it is longer than 120 characters (`models.MaxPublishingAuthority`)
or carries a control character, so it is always one line a phone can show.
A refused publish leaves the form a draft. Composing or editing a form can
set neither field: `Create` clears both and `Update` copies neither.

## Existing tables

`spec.Migrate` creates a table that is missing but never adds a column to
one that exists. `Provision` therefore adds any declared optional text
column a form table lacks, as `text default ''`, which is what a fresh
migration would have created. A missing column of any other type is an
error naming it rather than a silent gap.
