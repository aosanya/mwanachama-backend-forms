# Publishing a closed form's report (F32)

Member app decisions 210 and 212 (owner, 2026-10-21). Once a form is closed,
its report — what it collected, written up — is published to the groups the
publisher chooses (F39), and a resident in one of them, or in a group inside
one, can find it.

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
- **The groups are the publisher's to choose** (F39, member app decision
  245, amending 210). The optional body `chapter_ids` names one or more
  groups, any group of the agency rather than only the form's targets: a
  form sent to traders may be reported to traders and the finance
  committee. Forms does not know the agency's groups, so checking that each
  id is one is the host's. Blanks and repeats are dropped, in the order
  given. Omitted, the report goes to **every target** (the studio starts
  them ticked). The row stores the list as `chapter_ids` and its first entry
  as `chapter_id`; nothing chosen and no target is `ErrNoTarget` (409).
- Publishing again records the **next version** (1, 2, 3 …) and leaves the
  earlier rows alone; the `(form_id, version)` index is unique. Each publish
  also sets the form's `results_published_at`, in the same transaction.

Who may publish is not forms' to decide: it is the action gate the host
applies to `forms.results.publish` (member app Q18).

## Reading — `GET /forms/{form_id}/results/latest`

- Action `forms.results.read`; the caller is bound `from: caller`.
- Answers the newest version for a caller whose groups (the host's `Groups`
  resolver) include **any group the report was published to** (F39).
  A group left unchosen does not read it, even a target of the form.
- **A chosen group's inner groups read it too** (member app decision 246):
  the host's `Groups` names a caller's own groups and every group they sit
  inside (wakala-api WK167), so a resident of a sub-group matches the chosen
  group above them. Where the host's `Groups` names only a caller's own
  groups, the read falls back to the host's `Members` hook with
  `includes_descendants` set (the same seam F34's missed list uses): a
  caller among the members within a chosen group reads it (wakala-api
  WK155). Forms walks no tree itself, and a mount with no `Members` gets only
  the `Groups` match.
- A report published before F39 carries no `chapter_ids`; it keeps F34's
  rule and reaches every target of the form.
- A form with no report, a report published to none of the caller's groups,
  a caller with no session and a mount with no `Groups` all answer
  `ErrReportNotFound` (404) — the same answer as a form that does not exist,
  so the read cannot be used to learn which forms have reports.

Pinned by `routes/f32_publishing_results_test.go`, including
`TestAReportOnAFormAimedAtSeveralGroupsIsReadInEveryOne`, and
`routes/f39_report_to_chosen_groups_test.go`, including
`TestAReportReachesAnInnerGroupTheHostNamesOnlyAsMembersWithin`.
