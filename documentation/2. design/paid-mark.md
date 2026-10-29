# The paid mark (HS100)

A send already is the person's own record in the forms catalogue: one `submission` per form, person and group. The paid mark is two columns on it, `paid_at` and `paid_by`, so it sits beside the answers and not in them. A resend before the close rewrites the same submission row from what is stored, so it moves `edited_at` and leaves the mark alone; `hs100_paid_mark_test.go` pins that.

`MarkPaid(form, member, paid, markedBy)` sets or clears the mark. It answers `ErrSubmissionNotFound` for a person who sent nothing, since there is no record to mark. `paid_by` comes from the session (`from: caller`), never the body. The address is `PUT /forms/{form_id}/submissions/{member_id}/paid`, action `forms.submission.mark_paid`.

Reads need no new address. `GET /forms/{form_id}/submissions/mine` is the farmer's own entry and `GET /forms/{form_id}/submissions` is the whole list, both carrying `paid`, `paid_at` and `paid_by`.

This module does not know who the organiser is. The host gates `forms.submission.mark_paid` to the group coordinator and `forms.submission.read_mine` to the farmer. Nothing here takes a payment and nothing posts in a chat.

What remains outside this repo: wakala-api granting the action to the group coordinator role, the member app's "Not paid yet" / "Paid" line from `paid`, and the studio's list with its mark control.

The Group order no longer uses this mark (Hub Steward decision 90, HS296): who has paid there is a row in the Group order's Payments record, kept in the agency and added by the Group coordinator in the member app. The mark stays in this module, unchanged, for any other commitment that wants it.
