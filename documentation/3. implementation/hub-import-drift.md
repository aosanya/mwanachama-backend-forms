# Hub import drift

Where the hub's copy of the forms module (`projects/hub-steward/code/wakala-api/modules/forms`, read at hub commit `fed51ac0`) and this repository differed on 2026-11-19, and what was done about each difference. This repository is the custodian of the module; the hub is to import it and delete its copy, so every difference needed a disposition first.

## How it was measured

`scripts/hub-copy-diff.sh` (see [hub-copy-diff.md](hub-copy-diff.md)) rewrites the hub's imports, drops all comments and prints every Go file with `gofmt` before diffing. Before the merge it listed 25 files only in the hub, 98 only here and 40 that differ; after it, none only in the hub and 14 that differ. The hub had moved on by 21 commits since this repository last replayed it (the last replayed change was `f6160f9e`); `401d0cd4` (the report-text reads) had come from this repository in the first place.

## Result of the merge

The hub's code now lives here as well. After the merge the script lists no file that exists only in the hub. The files that still differ are all differences kept on purpose, in the third table below.

## Only in the hub, now here

| Hub change | Files here | Behaviour |
|---|---|---|
| `c912f0f8` correct a sent application | `correct_impl.go`, `correct_submission_test.go`, the `correct_submission` operation in `forms.operations.json` | A clerk replaces named answers on a sent application, also on a form that locks after sending. |
| `147d7729` decide a send | `decide_impl.go`, `decide_errors.go`, `decide_submission_test.go`, `routes/decide_submission_caller_test.go`, `models/submission.go` (`Decision`, `DecidedBy`, `DecidedAt`, `DecisionNote`), `models/answered.go` (`Decision`, `DecidedAt`), `answered_impl.go`, `my_submission_impl.go`, `standing_impl.go` | A clerk records approved or refused on an applicant's send; an applicant cannot decide their own; the applicant reads the decision back. |
| `dea62aed`, `ddb76c82` certificate and end day | `decide_impl.go`, `decide_errors.go`, `models/submission.go` (`CertificateRef`, `ValidUntil`) | A grant may carry a certificate number and an end day; both are refused on a refusal or when the day is past. |
| `8b09a295`, `882f3c1a` inspection | `inspect_impl.go`, `inspect_errors.go`, `inspect_submission_test.go`, `models/inspection.go`, `routes/inspect_submission_caller_test.go`, `models/submission.go` (`InspectionResult`, `InspectionNote`, `InspectedBy`, `InspectedAt`) | An inspector records a finding on a send; nobody inspects their own; the applicant's reads leave the finding out (`withoutFinding`). |
| `b706d7f6` inspector list | `inspector_list_impl.go`, `inspector_list_test.go`, `UseInspectorForms` on the manager | The sends of the permits whose inspector role the caller holds, through a seam the host fills. |
| `b3a79d78`, `4a66e903`, `dd97bc8d` a person's sends across forms | `person_submissions_impl.go`, `person_submissions_red_test.go`, `routes/person_sends_route.go`, `routes/person_submissions_red_test.go`, `models/submission.go` (`FormTitle`) | Staff list every send one person made, across forms; the route refuses a request that names no person when the caller is not authorised to be unnamed. |
| `8f9b5e31`, `3913f24e` record row on a send | `record_row_submission_test.go`, `recordRowID` through `SubmitAnswersFor`, `FileSubmission`, `ListSubmissions`, `send_keep.go`, `submission_impl.go`, `models/submission.go` (`RecordRowID`) | A send may name the business or trader row it is for; listing filters along the row. The owner check on the row is the host's. |
| `f643bc0f` forms reaching a person | `reaching_person_impl.go`, `reaching_person_test.go`, `ListReachingPerson` on the manager | The published forms that reach one person by the module's own reach rule. |
| `046f2eb1`, `b02bb2a8` approving above the limit | `approval_over_limit_test.go`, `approval_impl.go` (`ApprovalCover`, `ApprovalLimits` returns it), `models/form.go` and `models/approval.go` (`OverLimit`, `LimitInForce`), `form_impl.go` | A role may be allowed to approve above its limit; the approval and the form are flagged and the flag is cleared when the form goes back to draft. |
| `5e8b4726` time range answer | `models/answer.go` (`ValueTimeTo`), `models/validators.go`, `models/lands_in.go`, `models/lands_in_test.go`, `models/question.go` | A `time_range` answer with a start and an end, landed as `from 09:00 to 22:00`. |
| `bf5bfa28`, `be54802c`, `9c8131fe` address answer | `address_check_impl.go`, `address_ref_test.go`, `models/address_answer.go`, `models/address_answer_test.go`, `models/answer.go` (`PlaceRef`), `models/question.go` (`AllowFreeText`), `UseAddressCheck` on the manager, `testrun.go` | An `address` answer carries the words and a place reference; a reference the town does not hold is refused when the host passes a check. |
| `fed51ac0` | the same files | Shared answer rules for the time range and the place reference. |
| the operations above | `forms.operations.json`, `forms.blueprint.json`, `errors.go`, `routes/routes.go` (new sentinels) | New routes: reaching, a person's sends, inspectable sends, decide, inspect, correct. New error codes: invalid or self decision, grant on a refusal, invalid or past end day, invalid or self inspection. |

## Only here, kept

| Item | Disposition |
|---|---|
| The `AgencyApprovals` interface, `ProvisionOption`, `WithAgencyApprovals`, `Provision(db, s, opts...)` in `approvals_apart.go` and `provision.go`, and `memoryApprovalStore` in `approvals_clash_test.go` | Kept. The hub's file imports `wakalaserver/internal/approvals`, which Go forbids importing from another module. The hub's mount passes its store when it imports this module. |
| `ErrReportNotText` and its 422 in `errors.go`, `forms.operations.json` and `routes/routes.go` | Kept; see the owner questions. |
| `report_text_impl.go` as it stands here | Kept; see the owner questions. |
| The strict `HH:MM` time rule (`checkClockTime`, `ClockLayout` in `models/validators.go`, one line in `models/answer.go`) and its tests | Kept; see the owner questions. |
| `scripts/`, `documentation/`, the test files listed as `Only in backend` by the script | Kept; they are this repository's own tests and tooling. |

## Differing beyond the import path, kept

| File | Difference | Disposition |
|---|---|---|
| `models/schedule_cover_test.go`, `question_compose_cover_test.go`, `reach_delete_cover_test.go` | `weeklyOnMondays`, `publishForm` against the hub's `weekly`, `publish` | Kept; the names avoid a clash with helpers here. |
| `routes/mun1373_clerk_files_for_a_resident_test.go` | The hub's copy declares the route test helpers (`newTestManager`, `seedForm`, `callerMount`, `doAs`, `decodeAnswers`); here the same helpers live in the routes test files already | Kept; the test cases are identical. |
| `models/answer_validators_cover_test.go` | Cases for the strict time rule | Kept with the rule. |
| `forms.operations.json` | One caller argument carries a description in the hub and not here; the text of `read_result_text` names the 422 here | The hub's argument description was taken; the 422 text is kept. |
| `routes/routes.go` | Same sentinel map, formatted one per line, plus `ErrReportNotText` | Kept. |

## Hub wiring that is not part of the module

Not moved; each stays in the hub or is for the import step to pass in.

| Hub piece | What the import step does |
|---|---|
| `internal/agencies/registry.go` calls `UseAddressCheck`, `UseApprovalLimits` and `UseInspectorForms`; `approval_limits.go` returns `ApprovalCover` | Stays in the hub; it already matches the signatures kept here. |
| Passing the agency approvals store to `Provision` | New in the hub: `WithAgencyApprovals(store)` where the hub calls `Provision`. |
| The report text reader the host fills (`UseReportText`) | Stays in the hub. |
| The permission catalogue (`application.file`, `application.correct`, `application.inspect`, `application.decide`, `board.manage`, `report.read`) and the actions a resident role may hold | Stays in the hub. |
| The MCP tool lists and filters, and the roster pins (for example `EveryMountConfiguration`) | Stays in the hub; the pins follow the new route count when the hub imports. |
| The gateway tests in `internal/api/http` that drive these routes (resident decide, resident results and others) | Stay in the hub. |
| The owner check that a `record_row_id` names a row the person owns | Stays in the hub; the module stores and filters the id only. |

## Pinned counts changed by the union

`routes/table_test.go` pinned 55 addresses and now pins 61: `GET /forms/reaching`, `GET /submissions`, `GET /submissions/inspectable`, `PUT /forms/{form_id}/submissions/{member_id}/answers`, `PUT /forms/{form_id}/submissions/{member_id}/decision` and `PUT /forms/{form_id}/submissions/{member_id}/inspection` came with the hub's code. No test was loosened or skipped.

Calls in this repository's own tests to `ListSubmissions` and `SubmitAnswersFor` gained the empty record row argument the hub's signatures carry.

## Dependency on shared

`models/validators.go` imports `mwanachama-backend-shared/answerrules`. The first shared tag that holds it is `v0.3.0`; `v0.1.0` and `v0.2.0` do not. This repository's `go.mod` still points at a local copy of shared through a `replace`.

## Owner questions

1. The time answer here is strict (`07:05` only, [time-answers.md](time-answers.md)); the hub accepts `7:05`. The new `time_range` answer takes its rule from shared `answerrules`, which accepts a one-digit hour. Should a time range be as strict as a time, with the rule in `answerrules`?
2. The text of a report is read here with the host's reason hidden (502 with no reason, 422 for a file that is not text, a missing form answers as a missing report, ties in the list ordered by form id). The hub's copy of the same read carries the reason in the error and answers with a blank title for a missing form. Keeping the version here is the proposal; confirm it before the hub drops its copy.
