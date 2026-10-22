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

## The manager

The methods sit on the `Translator` interface, which the form manager
implements; a mount reaches it with a type assertion on `FormManager`.

- `PutTranslation` — write or replace one text. The form must exist.
- `ListTranslations` — every text of a form, reviewed or not, for review.
- `ReviewedTranslations(form, language)` — the reviewed texts only; what a
  read in that language overlays on the original.
- `TranslationLanguages(form)` — the languages with at least one reviewed
  text, sorted; what a read lists as available.

## Not yet built

- The form's own original language (a field on `form`).
- Reads that take a language: `get_form`, `list_questions`, `list_options`
  and the public-link questions overlaying the reviewed texts and answering
  the available languages.
- The operations in `forms.operations.json`, so wakala-api's WK177 can reach
  any of this over HTTP and MCP.
- A submission recording the language it was answered in.
- Checking that `subject_id` names a question or option of that form.
