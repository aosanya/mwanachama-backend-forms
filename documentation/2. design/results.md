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

## Finding and reading a report as a resident (SW615)

`forms.results.read` needs the form's id, and listing forms needs a manager
permission, so a resident had no way to find a report. Two more reads close
that, with no git permission involved.

| Tool | Route | Action | Hub permission |
| --- | --- | --- | --- |
| `forms_results_list_mine` | `GET /forms/reports/mine` | `forms.results.list_mine` | `report.read` |
| `forms_results_read_text` | `GET /forms/{form_id}/results/text` | `forms.results.read_text` | `report.read` |

`report.read` is the hub catalogue's resident-style code, separate from
`report.publish`. Forms declares only the action names; which role holds the
permission is the host's.

**List.** Each row is the form's id and title, the report's version, its
`published_at` and its `git_path`, newest published first, equal times by
form id. A form is listed once, by its **newest** version, and only when that
version was published to a group the caller is in or a group inside one (the
same reach as the read above). A newest version published to other groups
hides the older ones: the form is not listed, and an older version that did
reach the caller does not stand in for it. A caller in no group, a caller with
no session and a mount with no `Groups` list `[]`.

**Read text.** The same reach as `forms.results.read`, so a form with no
report, a report published elsewhere and a missing form all answer
`ErrReportNotFound` (404) with the same body. The text comes from the host's
reader, `UseReportText(ReportTextReader)`; see
`3. implementation/report-text-host-hook.md` for what the host provides.

- The text is cut at `MaxReportTextBytes` (200000) on a character boundary,
  never inside a multi-byte character. `size` is the whole file in bytes and
  `truncated` is true when the cut dropped anything.
- A mount with no reader, or a reader that returns an error, answers
  `ErrReportTextUnavailable` (502). The body is the fixed sentence "the
  report's text cannot be read now"; the host's reason is logged and never
  sent, because it can carry paths and repository names.
- A file that is not valid UTF-8 (a zip, a PDF) answers `ErrReportNotText`
  (422) instead of a text with replacement characters.
- The refusal for a caller outside the groups comes before the reader is
  called, so the host reads nothing for them.

Pinned by `routes/sw615_resident_finds_and_reads_a_report_test.go`.
