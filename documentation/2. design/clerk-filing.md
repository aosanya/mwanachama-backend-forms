# A clerk files for a resident

Hub issue MUN1373. A resident who has no sign-in brings an application to a desk, on paper or in the field, and a clerk files it for them.

## The operation

| | |
| --- | --- |
| Operation | `file_submission`, tool `forms_submission_file` |
| Address | `POST /forms/{form_id}/submissions/{member_id}/answers` |
| Action | `forms.submission.file` |
| Permission code | `application.file`, granted by the host to the clerk role; this module only declares the action |
| Manager | `FileSubmission(ctx, formID, memberID, answeringAs, language, periodKey, channel, filedBy string, answers []models.Answer) ([]models.Answer, error)` |

The address shares its path with the hub's `correct_submission` (a `PUT`) and with nothing in this repository; the verbs keep them apart.

Arguments: `form_id` and `member_id` from the path; query `as`, `language`, `period_key` as on `submit_answers`; query `channel`; `filed_by` from the caller, never from the request; body `answers`. The reply is the one `submit_answers` gives, every answer carrying `send_id`.

## One pipeline

`FileSubmission` and `SubmitAnswersFor` both run `submitAnswers`. A resident answering for themselves is the same call with the resident as both the person and the collector, so `submit_answers` behaves as it did.

Everything that guards an answer guards the named person, not the clerk:

- the form must be open and inside its window;
- the person must be reached by a group the form was aimed at, checked through `reaches` for `member_id`; the clerk's own groups play no part, and a mount with `ReachedOnly` does not narrow the call to the clerk. A person who is not reached is refused with `ErrInvalidAnswer` (400);
- every answer is validated, and every required prompt must be answered, before any is written;
- the consent gate is asked about `member_id`, and a clerk with no consent of their own is not stopped.

## Channel and collector

The send is stamped `channel` with `paper` or `field`. Absent means `paper`. Any other value, including `app` and `link`, which the module assigns itself, is refused with `ErrInvalidChannel` (400). The send's `collector` is the clerk and `collected_at` is the moment of filing.

A filing with no caller is refused with `ErrInvalidAnswer` (400), so a mount that supplies no caller cannot file under an empty clerk.

## Filing again

Filing again for the same person, form, group and period edits the send already kept, exactly as answering again does: one send, the same `send_id`, `edited_at` set, and the collector and channel of the last filing. A form that locks after sending refuses the second filing with `ErrLockedAfterSend`; correcting such a send is the hub's `correct_submission`.

## A person with no consent record

`consent.Require` returns nil when the mount wired no gate, so a module mounted without one files for anyone reached. With a gate wired, the gate's own rule decides for the named person: the hub's gate passes everyone when the instance does not require consent, and otherwise answers `ErrConsentRequired` (403) for a person with no consent record and `ErrConsentWithdrawn` (403) after a withdrawal. Nothing is filed in either case. A resident with no sign-in cannot agree in the app, so an instance that requires consent can file for them only after their consent is recorded some other way.

## Not carried here

`record_row_id` on a send exists only in the hub copy (blueprint column on `submission`, query argument on `submit_answers` and this operation). It is carried when the hub's record-row work is replayed into this repository.
