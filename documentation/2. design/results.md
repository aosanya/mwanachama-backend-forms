# Publishing a closed form's report (F32)

Member app decisions 210 and 212 (owner, 2026-10-21). Once a form is closed,
its report — what it collected, written up — is published to the group it was
aimed at, and a resident in that group can find it.

## What forms stores, and what it does not

The report itself is not stored here. The host writes it to a git repository
and hands forms two things: **`git_path`**, where it sits, and **`git_commit`**,
the commit that holds this version. Forms records that reference as a
`result_report` row, with the group, who published and when. Answers stay
forms' own; the records built from them live in catalog (CAT48).

## Publishing — `POST /forms/{form_id}/results`

- Action `forms.results.publish`, status 201.
- Only a **closed** form is accepted; anything else is `ErrNotClosed` (409).
- `git_path`, `git_commit` and the publisher are required; a missing one is
  `ErrInvalidReport` (400).
- `published_by` is bound `from: caller`, so a body naming somebody else is
  overwritten.
- The group is never the body's. The row's `chapter_id` is the form's
  **first target**; a form with no target is `ErrNoTarget` (409).
- Publishing again records the **next version** (1, 2, 3 …) and leaves the
  earlier rows alone; the `(form_id, version)` index is unique. Each publish
  also sets the form's `results_published_at`, in the same transaction.

Who may publish is not forms' to decide: it is the action gate the host
applies to `forms.results.publish` (member app Q18).

## Reading — `GET /forms/{form_id}/results/latest`

- Action `forms.results.read`; the caller is bound `from: caller`.
- Answers the newest version for a caller whose groups (the host's `Groups`
  resolver) include **any of the form's targets** (F34, member app decision
  236): a report on a form aimed at several groups reaches every one of them.
  Targets are fixed once a form leaves draft, so the set a report reaches
  cannot move after it is published. Reach into a target's sub-groups is the
  host's `Groups` resolver's (wakala-api WK167), not forms'.
- A form with no report, a report published to none of the caller's groups,
  a caller with no session and a mount with no `Groups` all answer
  `ErrReportNotFound` (404) — the same answer as a form that does not exist,
  so the read cannot be used to learn which forms have reports.

Pinned by `routes/f32_publishing_results_test.go`, including
`TestAReportOnAFormAimedAtSeveralGroupsIsReadInEveryOne`.
