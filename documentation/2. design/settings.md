# Forms' settings (F42)

Member app decisions 280, 281 and 398: each module stores and checks its own
settings, in a definition layer and an override layer. Forms declares two, in
`forms.blueprint.json`'s `settings` block, through shared's S64 engine
([declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md)).

| Setting | Type | Module default | What it does |
| --- | --- | --- | --- |
| `approval_required` | bool | `false` | Whether a form must be submitted and approved before it is published ([approval.md](approval.md), F33). |
| `form_types` | json | `[]` | The kinds of form a deployment composes: a list of `{name, audience, collection_mode}`, each the audience and collection mode a new form of that kind starts with. |

Both defaults keep the behaviour from before F42: no approval step, and no
named kinds, so every form starts with the field defaults (`member`, `self`).

## How the manager reads them

`Provision` runs `spec.Migrate`, which creates the three settings tables
(`spec_settings_definition`, `_override`, `_change`) in forms' own table set
because the blueprint declares settings. `NewFormManager` resolves both
settings once through `spec.ResolveStored`: an override wins over the
definition, the definition over the module default, and a stored value that no
longer fits its shape is passed over by the engine.

- `approval_required` seeds the same flag `UseApprovalRequired` sets. A host
  that still calls `UseApprovalRequired` after construction overrides the
  resolved value, exactly as before.
- `form_types` is held to `models.ParseFormTypes`: every kind is named once,
  its audience is `member` or `public`, its collection mode `self` or
  `interviewer`, and the pair passes `models.ValidateAudienceCollection` (a
  member audience is never interviewed). A value that breaks any of these
  refuses the manager at construction. `FormManager.FormTypes()` returns the
  list.

Resolution happens at construction. A setting changed while a manager is
serving is read by the next manager built; nothing re-reads it per request.

## What is not built

- **Nothing applies a form type to a new form yet.** The module stores and
  checks the list; choosing a type when composing, and which defaults it
  fills, is not decided by F42 and is not invented here.
- **No route or tool writes a setting.** The host writes the definition layer
  (`spec.PutDefinitionSettings`) and an override (`spec.StoreOverride`), and
  the change log is `spec.SettingChanges`.
- **A stale override after a republish is an open owner question** (S64).
  Forms neither clears nor keeps one deliberately; it takes what the engine
  resolves.
