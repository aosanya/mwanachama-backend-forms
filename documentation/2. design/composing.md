# Composing a form's questions (F89)

Member app decision 215: while a form is being composed the operator edits
each question, removes questions and reorders them, and each change saves as
it is made.

| Operation | Address | Action |
| --- | --- | --- |
| `remove_question` | `DELETE /forms/{form_id}/questions/{question_id}` | `forms.question.delete` |
| `move_question` | `POST /forms/{form_id}/questions/{question_id}/position`, body `{"position": n}` | `forms.question.move` |

- **Draft only.** Both refuse with `ErrNotDraft` (409) once the form has left
  draft, the same rule `add_question` and `update_question` keep. A question
  addressed under a form it does not belong to is `ErrQuestionNotFound`.
- **Ordinals stay `1..n` with no gaps.** A removal deletes the question and
  its choices in one transaction and renumbers the rest in their order; a move
  takes the question out, puts it at `position` (counted from 1, a position
  outside `1..n` is `ErrInvalidQuestion`, 400) and renumbers every question
  of the form. `add_question` keeps appending at `count + 1`, which stays
  correct because there is never a gap. `move_question` answers the form's
  questions in their new order.
- A draft question has exactly one version, so the version chain is not
  touched: no draft question is ever superseded.

## Left out

- Removing or reordering after publish. A published question has answers
  against it and versions through `version_question`; what removing or moving
  one would mean is a product question nobody has answered.
- A removed question's translations (`translations.md`) are not deleted. They
  stop being reachable through the question; whether they should be removed
  with it is not decided here.
