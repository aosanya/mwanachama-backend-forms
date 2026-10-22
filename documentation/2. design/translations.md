# Translations (F41, first slice)

Member app decisions 256 and 257: a form, its questions and its options carry
their text in other languages, each text marked reviewed or not, and only a
reviewed text is served.

## The object

`translation` is its own declared object rather than a column on `form`,
`question` or `question_option`, so a language is added without a migration
and the three objects it renders stay untouched. One row is one text:

| field | meaning |
| --- | --- |
| `form_id` | the instrument the text belongs to, so a form's texts read in one query |
| `subject_role` | `form`, `question` or `question_option` |
| `subject_id` | the object rendered |
| `field` | which of its texts: `title`, `description`, `prompt`, `label` |
| `language` | a language tag |
| `text` | the text in that language |
| `reviewed` | whether it may be served |

`(subject_id, field, language)` is unique: putting the same text again
replaces it. A correction arrives with `reviewed` as the caller sends it, so
a changed text sent unreviewed stops being served until it is reviewed again.
A put naming a row held by another form or another subject role is refused.

## The original language

`form.language` is the tag the form was written in: what its own title,
description, prompts and choices are in, and what a reader is shown when no
reviewed text in their language exists. It is optional and empty when the
composer named none. Whether it should be required, and whether it should
count among the languages a read lists as available, is not yet decided.

## The manager

The methods sit on the `Translator` interface, which `FormManager` embeds, so
every manager a mount holds carries them and a shape built from
`struct{ FormManager }` still dispatches the four operations below.

- `PutTranslation` — write or replace one text. The form must exist.
- `ListTranslations` — every text of a form, reviewed or not, for review.
- `ReviewedTranslations(form, language)` — the reviewed texts only; what a
  read in that language overlays on the original.
- `TranslationLanguages(form)` — the languages with at least one reviewed
  text, sorted; what a read lists as available.

## The operations

Declared in `forms.operations.json`, so they reach HTTP and MCP alike, each
gated by its own action:

| operation | address | action |
| --- | --- | --- |
| `put_translation` | `PUT /forms/{form_id}/translations` | `forms.translation.put` |
| `list_translations` | `GET /forms/{form_id}/translations` | `forms.translation.list` |
| `list_reviewed_translations` | `GET /forms/{form_id}/translations/reviewed?language=` | `forms.translation.list_reviewed` |
| `list_translation_languages` | `GET /forms/{form_id}/languages` | `forms.translation.languages` |

The address outranks the body: a put names its form by the path, and a body
claiming another `form_id` is overwritten. Writing and reviewing are one
action today; whether reviewing should be a separate grant from drafting is
open (decision 256 says the operator reviews, not who may draft).

## The language a send was answered in

A submission carries `language`, the tag the sender answered in.
`submit_answers` takes it as the `language` query argument and
`SubmitAnswersIn` records it on the send; `SubmitAnswers` and
`SubmitAnswersAs` record none. A resend records its own language, so the
send says what the latest answers were given in, while `submitted_at` keeps
the first send's time. Empty means the send named none.

The tag is recorded as sent and not checked against the form's languages:
which languages a form offers depends on whether its original language counts
among them, which is not yet decided.

## Not yet built

- Reads that take a language: `get_form`, `list_questions`, `list_options`
  and the public-link questions overlaying the reviewed texts and answering
  the available languages.
- Checking that `subject_id` names a question or option of that form.
