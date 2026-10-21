# Approval is each agency's choice (F33)

Member app decision 216 (owner, 2026-10-21). Whether a form has to be
cleared by somebody before it is published is a per-instance setting,
`approval_required`, and it is **off by default**.

## The setting

The host turns it on per mounted instance with
`FormManager.UseApprovalRequired(true)`, beside `UseGroups` and `UseNames`.
A manager never told anything behaves exactly as it did before F33.

## When it is on

- **`publish` refuses a form that has not been cleared.** A draft answers
  `ErrNotApproved` (409) and stays a draft. A submitted form still answers
  `ErrAwaitingApproval`, and an approved one publishes as before.
- **`approvals/approve` refuses the form's own author** with
  `ErrSelfApproval` (403), and the round stays open. The author is the
  form's `created_by` (F22) or whoever submitted it for this round; either
  of them clearing it would make the second pair of eyes the first.
- **Refusing carries a reason and returns the form to draft.** A refusal
  with a blank note is `ErrMissingNote`; a refusal with one sends the form
  back to draft and records the note on the round. This was already true
  before F33 and is unchanged.

## When it is off

Approval is never asked: a draft publishes directly, and a form that is
submitted anyway can be cleared by anyone, its author included.

## The approver is the caller

`approve_form` and `refuse_form` bind `approver_id` with
`{"from": "caller"}`. Before F33 it was body text, which would have made
the self-approval check a field the author fills in. A body naming another
approver is overwritten, the same rule `actor_id`, `member_id`,
`published_by` and `created_by` already follow.
