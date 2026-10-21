# A public link's declared group is a real one (F35)

Member app decision 236: a consultation round's public link asks the
respondent for their ward, checked against the real wards. This module does
not know what a ward is, or which groups exist, so the host says.

## The host names the groups a form offers

`UseDeclarable(func(ctx, formID) ([]string, error))` gives the manager the
groups a respondent may declare on one public form. `DeclareChapter` asks it
after the link has resolved and before anything is written:

- A link that does not resolve still answers `ErrLinkNotFound` first, so the
  check never becomes a way to probe a key (the refusal still names no
  reason).
- A `declared_chapter_id` outside the offered list is `ErrInvalidReference`
  (400). This refusal does name its reason, because the respondent picked
  from a list the form itself gave them.
- An empty `declared_chapter_id` is not checked: the respondent typed
  `declared_text` instead, where no group fitted.
- A manager given no resolver checks nothing and records any group, as it
  did before F35, so a host that has not wired one keeps working.

## Declarations are listed and counted per group

`ListDeclarations(ctx, formID)` returns a form's declarations, oldest first.
`CountDeclarations(ctx, formID)` returns one `{chapter_id, count}` per
declared group, sorted by group; declarations made by free text alone count
under the empty `chapter_id`.

Both are manager methods only. No address in `forms.operations.json` serves
them yet; exposing them needs an action id and a grant, which is the host's
next step (wakala-api WK167).
