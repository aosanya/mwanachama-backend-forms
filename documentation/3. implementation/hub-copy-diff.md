# Hub copy diff

`mwanachama-backend-forms` is the custodian of the forms module. The hub (`projects/hub-steward/code/wakala-api/modules/forms`) holds a copy made at hub commit f871f83c and changed it afterwards. Every change made in that copy exists here except the one named below, so the two differ by the import path only (hub issue F88, the forms half of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-forms]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/forms`, relative to this repository's parent folder. The exit code is 1 when the trees differ. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-git` (G78), generalised: the module path is read from `go.mod` and the hub folder's name from the last element of the path given.

## What it does

1. Copies the `*.go` and `*.json` files of the hub folder, and the same kinds from this repository, into two temporary folders. `cmd/`, `documentation/`, `scripts/` and `.git/` are left out.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/` and `wakalaserver/modules/forms` becomes `github.com/aosanya/mwanachama-backend-forms`.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub`.

## Result of 2026-11-16

Every non-test file is identical after the rewrite except two, both because of one open owner decision:

| Difference | Reason |
|---|---|
| Only in the hub: `approvals_apart.go`, and the two hunks of `provision.go` that call it | Hub commit 4df627b4 (an agency's approvals store and the forms module no longer share a table). The file imports `wakalaserver/internal/approvals`, the hub's agency approvals store, which no backend repository holds, so this public repository cannot import it. The owner picks where the store lives; the options are in the hub's `documentation/2. design/forms-approvals-apart.md`. |
| Only in the hub: `approvals_clash_test.go` | The test of the same commit. Its helpers already live here in `clash_db_test.go`. |

Test files differ in four places, all name changes made here to avoid a clash with a helper this repository already had: `weekly` is `weeklyOnMondays` in `models/schedule_cover_test.go`, and `publish` is `publishForm` in `reach_delete_cover_test.go` and `question_compose_cover_test.go`.

Every other line of the output is `Only in backend`: test files this repository had from before the copy, and tests written here for hub commits whose own tests live in the hub's gateway (`internal/api/http`), which is not part of this module:

| Test here | Hub commit it proves |
|---|---|
| `routes/stranger_link_view_test.go` | 9db3665b |
| `routes/resident_my_submission_view_test.go` | 5762d51f |
| `routes/staff_results_view_test.go` and the narrowed read in `routes/resident_results_view_test.go` | 080f5664 |
| `routes/list_window_test.go` | e97a230a |
| `two_process_send_test.go` | 992ba583: two managers on one sqlite file, and on Postgres when `POSTGRES_URL` is set, send together as two groups and keep one voice; the hub's proof drives two whole servers |

## The hub commits carried over

One commit here per hub commit, each naming the hub hash: baf83b7a, ec77c144, d0ddefd7, b1f9348c, a37f9023, 997c7b76, 638e043f, 20050ce0, 782efe38, 82523cbb, 637cb197, 2e3a7bb7, 50ea2e52, 6e030dfc, 3f2ffa80, 5e67327c, c737a4fe, 2303bf4f, 9fcdb045, c977a8ae, 1b035f31, 832d4470, 27df7d13, ad35d462, 7d239f86, c5fc8c96, 9db3665b, 6ea02a12, b168c25f, ef148dbb, 5c37d585, 1fa9e664, 9b603ee4, 6ef0dde9, 5762d51f, 7bfec797, 5c274fc4, e97a230a, 080f5664, 992ba583, f6160f9e.

Not carried: 4df627b4, waiting on the owner. When it is carried, `approvals_apart.go` and the two `provision.go` hunks are the whole difference and the diff reads import path only.

Commits that changed several modules were replayed for the forms folder alone. Files a commit touched lost their comments (the repository rule is no comments in code). Where a hub commit changed what the backend's own tests asserted (public registration, the latest-results read), those tests were updated in the same commit.

## Clean export run

`go build ./...`, `go vet ./...` and `go test -race -p 2 -timeout 40m ./...` run green on a `git archive` export of this repository's HEAD, with the shared repository's HEAD exported beside it and a `replace` of `mwanachama-backend-shared` pointing at that export, in the scratch copy only. The Postgres cases skip without `POSTGRES_URL`.
