# A thing's stable identity

`FormManager` embeds the shared `identity.Owner` (shared S56), so a host that resolves a manager per instance already holds the owner for it. No board row named this; it was opened from the pattern agency set (AG77 to AG79).

## The raw

`StableIdentity(ctx, object, id)` answers `forms/<instance>/<uuid>` for the two objects other things point at: a `form` and a `question`. The instance is the spec's instance, the one that prefixes the tables.

A form keeps its row id for life, so its identity is that id.

A question does not: `VersionQuestion` writes a successor row with a new id and `supersedes_question_id` pointing back. The identity of a question is the id of the first version in its chain, so every version answers the same raw, and the row id of a version alone is not an identity.

`Resolve(ctx, raw)` tries a form first, then a question. For a question it answers the **current** version: the end of the supersedes chain that starts at the raw's id. A raw built from a later version's row id is `ErrNotFound`, so a thing has one raw only.

## What it answers

- `identity.ErrInvalidRaw` for a string that does not parse;
- `identity.ErrNotOwned` for another module's raw or another instance's;
- `identity.ErrNotFound` for an id no form or question chain holds, including a form deleted while it was a draft and the questions it took with it;
- `ErrNoStableIdentity` from `StableIdentity` for any other object.

## Left without an identity

Options are re-written for each version, targets and approvals belong to a form and have no operation that names them across modules, a public link is addressed by its key, and answers and submissions are a respondent's record. Giving any of them an identity is an owner decision, not a gap in the mechanism.

## The `refers` marks

`form_id` carries `"refers": "forms.form"` and `question_id` `"refers": "forms.question"` wherever the argument is bound (shared S57), so `Spec.References()` lists them. Not marked: the three `form_id` segments that are `ignored` (part of the address, nothing bound), `member_id` and the caller arguments (another module's ids), `target_id`, `send_id`, `key` and `period_key`.

An old version's `question_id` is still a reference: `StableIdentity` maps it to the chain's raw.

## The conversation

A tool's `Handle` takes the MCP `_meta` (shared S58) and puts `conversation_id` on the context the manager sees. This repo registers no MCP server, so there is no adapter to change; `routes/tool_conversation_meta_test.go` pins that `routes.BuildGatedTools` passes it through, including the reached-only narrowing, and a host that still calls `Tool.Invoke` passes none.

Tests: `identity_owner_test.go`, `operations_refers_test.go`, `routes/tool_conversation_meta_test.go`.
