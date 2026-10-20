# Design

The objects are declared, not written: `forms.blueprint.json` names every
object, field and index, `forms.forms.json` names what this domain calls
them and where they land, and `forms.operations.json` is the route table.
There are no row structs and no `AutoMigrate`. The format is documented
once, in
[declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md)
and
[dispatcher.md](../../../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md).

- [storage.md](storage.md) — what this module stores, and the decisions a
  reader cannot recover from the code: why timestamps are fixed-width
  strings, why nothing here holds a foreign key, and the three rules that
  have to live in SQL.
- [answers.md](answers.md) — whose an answer is: why `member_id` is bound
  from the caller, why reading everyone's answers is its own action, and
  why an empty send still counts as answered.
- [waiting.md](waiting.md) — the member's read, `GET /forms/waiting`:
  the open forms that reached the caller's groups, and the resolver the
  host supplies so those groups never come from the request.
- [answered.md](answered.md) — the member's other read,
  `GET /forms/answered`: every form the caller answered, open or closed,
  newest sent first, bound to the caller.
- [window.md](window.md) — whether a form keeps its `opens_at`/`closes_at`
  window: `window_advisory`, why it is named for the exception, and what
  forms written before it get.
- [reached.md](reached.md) — a resident reads and answers only the forms
  sent to their groups: `Mount.ReachedOnly`, and why a form that did not
  reach them answers exactly as a missing one.
- [claimed-link.md](claimed-link.md) — a signed-in person who answered
  through a public link claims the token they answered under, and may then
  read and answer that one form though it reached none of their groups.
- [link-questions.md](link-questions.md) — a public link's questions and
  their options, read without an account: the fourth anonymous action, why
  only current versions, and why every miss answers as `not found`.
- [publisher.md](publisher.md) — who published a form and on whose
  behalf: the caller's directory name through the `Names` resolver, and the
  authority from the publish body; and who composed it, `created_by` bound
  from the caller.
- [validators.md](validators.md) — what a question declares about the
  answers it accepts (required, number, date and selection bounds), why a
  bound is nullable, and how a submit is refused naming the question.
- [todo_details/FMD-001.md](todo_details/FMD-001.md) — the decision record
  for the conversion onto the declared-domain format, including what the
  plan assumed that turned out not to be true.
- [todo.md](todo.md) — the design board (`FMD-XXX`).

## Tables

`<instance>_forms_<object>`: `forms`, `questions`, `question_options`,
`targets`, `approvals`, `propagation`, `public_links`, `respondents`,
`declarations`, `answers`, `submissions`. A mount chooses only the instance, through
`SpecFor`; `Provision` moves a pre-spec table set onto these names before
creating what is missing.

## Topology

```text
Form ──has──────────► Question ──has──────► QuestionOption
Form ──has──────────► Target        (member-audience forms only)
Form ──has──────────► Approval      (one open at a time)
Form ──has──────────► Propagation   (one per Target, after Publish)
Form ──has──────────► PublicLink    (public-audience forms only, one, after Publish)
Respondent ──answers──► Answer ──answers──► Question
Respondent ──declares─► Declaration ──about──► Form
Respondent ──sends────► Submission  ──to─────► Form   (one per member, even with no answers)
```

A Question is superseded rather than edited once its form is published, so
several rows may describe one position, chained by
`supersedes_question_id`. Only the row nothing supersedes may be superseded
again.

## Where the rules live

- **Declared, and read off the spec** (`validate.go`): a required field is
  present, and an enum value is one the spec permits. A domain that adds a
  sixth status gets it enforced with no Go change.
- **In Go, beside the type they are about**, because no spec can state
  them: `ValidateWindow` (a closing time falls after an opening one),
  `ValidateAudienceCollection` (a member audience is never interviewed),
  `Form.OutsideWindow` (an answer arrives inside a window the form keeps),
  `ValidateAnswer` (a value fits the shape its own question asks for and
  the bounds it declares), `ValidateValidators` (those bounds can be met at
  all, see [validators.md](validators.md)), and
  the lifecycle transitions themselves.
- **In SQL** — the three in [storage.md](storage.md), Postgres only, with a
  Go rule beside each that the SQLite unit tests exercise.

## Not carried over from the extraction

No attribute catalog: nothing here is a free-form bag, every field is
fixed-shape. No custody or act-log writing on `AddOption`/`VersionQuestion`
— composing an audit event is the mounting process's job, and
`mwanachama-backend-api-gateway`'s `formsadapter` is where it happens.
