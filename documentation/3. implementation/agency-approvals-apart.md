# An agency's approvals apart from forms (hub 4df627b4, option B)

An agency keeps its own approvals in a table named `<agency>_approvals`. The forms module has an approval object whose legacy table name is the same. When both ran against one database, `renameLegacy` renamed the agency's table into forms, and the agency's approvals were swallowed into the forms approval table.

Hub commit 4df627b4 fixed it in two parts, and both are here.

1. `renameLegacy` leaves an agency-shaped table alone. A table is agency-shaped when it has the columns `kind` and `summary` (`holdsAgencyApprovals`). A forms-shaped legacy table is still renamed into forms.
2. `Provision` gives back to the agency's store any agency approval already swallowed into the forms approval table, in one transaction and only the ids the store lacks (`giveBackAgencyApprovals`, `moveAgencyApprovals`).

## Why an interface (owner's choice B, 2026-11-16)

The hub's version imports `wakalaserver/internal/approvals`, the agency approvals store, for two things: the table name of an agency's store and a call that makes sure that table exists. No public repository holds the store and Go forbids importing another module's `internal/`, so forms declares what it needs:

```
type AgencyApprovals interface {
	TableFor(agency string) string
	Open(db *gorm.DB, agency string) error
}
```

The mount passes it to provision as an option:

```
forms.Provision(db, spec, forms.WithAgencyApprovals(store))
```

`Provision` takes the options as a variadic argument, so a call without one still compiles.

## No store passed

If the forms approval table holds agency approvals and no store was passed, `Provision` fails with `forms: <table> holds agency approvals and no agency approvals store was passed to Provision`. It never skips quietly, so a mount that forgot the option is found at start-up and the swallowed rows stay where they are. When nothing was swallowed, no store is needed.

## What each mount does

Each API wraps its own approvals store in two one-line methods: `TableFor` forwards to the store's table name and `Open` calls the store's open function and drops the store it returns. The hub's mount (`internal/agencies/registry.go`, the call at `mwanachamaforms.Provision`) and the private API's registry pass it when the hub switches to this module (ISS-105).

## Tests

`approvals_clash_test.go` is the hub's file ported to an in-memory stand-in for the store (a plain table with the store's columns). The real store is checked by one small test in each API. It adds the case of no store passed.

## One more line of the hub commit

The same hub commit adds `form_member_slot` to `retiredIndexes`, so its one-per-person index is dropped on provision like the other three.
