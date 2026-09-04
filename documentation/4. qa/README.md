# QA

Test coverage and results for `mwanachama-backend-forms`. See this repo's
root and `routes/` for the actual test files (`*_test.go`); `go test ./...`
runs the sqlite-in-memory-backed unit suite with no database required, and
`postgres_integration_test.go` additionally exercises the real Postgres
wiring — including the `syncConstraints` trigger/CHECK/partial index — when
`POSTGRES_URL` is set.
