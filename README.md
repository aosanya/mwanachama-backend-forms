# mwanachama-backend-forms

Forms and surveys: build a form, target and approve it, publish it by link, and collect declarations and answers.

A Go library whose objects are declared in `forms.blueprint.json` and stored through
`mwanachama-backend-shared/specstore`. It is imported directly by whatever
mounts it (`mwanachama-wakala-api`); there is no separate service to run.

## Use

```sh
go get github.com/aosanya/mwanachama-backend-forms
```

`SpecFor(instance)` returns the default domain (`forms.forms.json`) for a host's
instance, `Provision(db, s)` creates its tables, and `NewFormManager(db, s)`
returns the manager that the rest of your code calls. The `routes/` package
builds the REST routes and MCP tools from `forms.operations.json`.

## Test

```sh
go test ./...
```

Unit tests run against in-memory SQLite. `postgres_integration_test.go` runs
against a real Postgres only when `POSTGRES_URL` is set.

## Licence

Apache-2.0. See [LICENSE](LICENSE). Design notes and the task board are in
[documentation/](documentation/).
