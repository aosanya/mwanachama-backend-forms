# The host hook that reads a report's text

Design: `2. design/results.md`, "Finding and reading a report as a resident".

## What a host provides

`forms_results_read_text` returns a report's text, but forms never opens the
repository that holds it. The host hands the manager a reader:

    manager.UseReportText(func(ctx context.Context, gitPath, gitCommit string) (string, error) { ... })

- It receives the report row's `git_path` and `git_commit` and returns the
  whole file as a string. Forms cuts it at 200000 bytes after it arrives, so a
  very large file is held in memory for the request.
- It is only called after the caller is known to be in a group the report was
  published to.
- Any error it returns becomes `ErrReportTextUnavailable` (502) with a fixed
  sentence. The error itself is written to the log with the form id and path
  and never reaches the response.
- Text that is not valid UTF-8 is refused by forms with `ErrReportNotText`
  (422); the reader does not need to check.
- With no reader set, the read answers 502. The list needs no reader.

In the hub, the reader lives in wakala-api (`internal/agencies/report_text.go`)
and the permission is `report.read` in that module's `permissions_catalog.json`.

## Open questions for the owner

These are questions, not decisions. Nothing here has been changed.

1. Should `git_path` be checked when a report is published? Today
   `PublishResults` checks only that it is plain text, so `../../x` and a moving
   ref such as `main` are stored and handed to the reader. What the reader can
   reach is left to the host.
2. Should one unreadable group fail the whole list? Today a lookup error for
   any group on any form's newest report returns a 500 to every caller.
3. Should the list be bounded or paged? It reads every report row of the
   instance, then looks up groups per form, so its cost grows with every
   publication.
4. Should a published report be withdrawable? There is no unpublish; a wrong
   report stays readable until a newer version goes to other groups, and
   publishing to no group is refused with `ErrNoTarget`.
