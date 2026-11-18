# What a host reader of a report's text can reach

Written 2026-11-18 for SW615. Read-only review: no probe was run, because the
hub has no reader yet (`UseReportText` is not called anywhere in
`projects/hub-steward/code/wakala-api`). These are the properties a reader built
on the hub's git manager inherits.

## What the library hands the reader

- `PublishResults` stores `git_path` and `git_commit` after a trim and a
  plain-text check only (`results_impl.go:15-24`).
- `ReadResultText` passes both to the reader unchanged, with no repository id
  (`report_text_impl.go`).

## What the git manager does with them

- **The commit is ignored.** The manager's file read is
  `ReadFile(ctx, branchID, path)` (`modules/git/git_impl_fileops.go:349`). It
  reads the branch's current head and takes no commit. A report rewritten after
  it was published is served in its newest form, and `git_commit` is stored and
  never used.
- **A `../` path does not escape.** `findBlobAtCommit` compares the path as an
  exact string against the stored blob paths, so an unknown path gives
  `ErrFileNotFound`. `ReadFile` does not trim a leading slash, though the write
  and delete paths do.
- **File bytes come from the blob row or the recorded clone**, not from a
  location the caller names.

## Which repository

- The caller of the manager names the repository by branch id. The one
  precedent, `forms_results_files`, takes the default branch of the repository
  named `reports`.
- Instances are separated by `Registry.GitManager(ctx, instanceID)`. The stored
  report does not record a repository.

## What that means

- Another instance's repository is not reachable if the reader uses the
  caller's own instance manager.
- Any file in the `reports` repository whose exact path a `report.publish`
  holder writes into `git_path` is readable by the groups the report is
  published to. That includes another form's report and files that are not
  reports.
- The git read tools are permission-gated, but a reader that calls `ReadFile`
  directly bypasses that gate. The gates that apply are `report.read` and the
  check that the newest report was published to one of the caller's groups.
- A resident gets one file's text per published report, cut at 200000 bytes,
  with no listing, history or other git read.

## Open for the owner

1. Should `git_path` be checked at publish (shape, a `<form_id>/` prefix, no
   `..`)?
2. Must `git_commit` be honoured? That needs a read at a commit in the git
   manager, which has none today.
