# Approval is each agency's choice (F33)

Member app decision 216 (owner, 2026-10-21). Whether a form has to be
cleared by somebody before it is published is a per-instance setting,
`approval_required`, and it is **off by default**.

## The setting

The host turns it on per mounted instance with
`FormManager.UseApprovalRequired(true)`, beside `UseGroups` and `UseNames`.
A manager never told anything behaves exactly as it did before F33.

The flag is an `atomic.Bool` on the manager, so a host may call
`UseApprovalRequired` on a manager that is already serving requests: a
publish, an approval or a refusal reading it at the same instant sees the old
value or the new one, never a torn read. `go test -race` pins it with
`TestApprovalRequiredCanBeSetWhilePublishesReadIt`.

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

There is no approval step (F90, member app decision 385): a draft publishes
directly, and `submit` refuses with `ErrApprovalNotRequired` (409), leaving the
form a draft. With approval on, the author never clears their own form on
either `approve` or `approve_slot` (decision 215). Both are pinned in
`f33_approval_required_test.go`.

## The approver is the caller

`approve_form` and `refuse_form` bind `approver_id` with
`{"from": "caller"}`. Before F33 it was body text, which would have made
the self-approval check a field the author fills in. A body naming another
approver is overwritten, the same rule `actor_id`, `member_id`,
`published_by` and `created_by` already follow.

## Several approvals before a round clears (F45)

A form may declare `approval_roles`: the distinct approval slots a round must
have granted before the form is cleared to publish. Empty, which every existing
form is, keeps the behaviour above exactly: one plain approval clears it.

- `POST /forms/{form_id}/approvals/approve-slot` (`forms.approval.approve_slot`)
  grants one slot, named in the body, inside the open round. The approver comes
  from the caller.
- A grant is its own decided `approval` row carrying `slot`, filed under the
  round's `requested_at`. Because it is decided at once it never counts as an
  open round, so the one-open-approval index is unchanged.
- An unknown slot is `ErrUnknownSlot` (400); a slot already granted in the round
  is `ErrSlotGranted` (409); an approver who already granted another slot of the
  round is `ErrApproverHoldsSlot` (409); the author is refused as before when
  approval is required.
- The form stays `submitted`, so publishing still answers
  `ErrAwaitingApproval`, until the last slot is granted; that grant closes the
  round as approved in the same transaction.
- A plain `approve` on a form that declares slots is `ErrSlotRequired` (409).
  `approve-slot` on a form with no slots is a plain approval.
- A refusal or a withdrawal closes the round as before; the grants it held are
  left behind as history, and the next round starts with none.

Who may name the slots, and which grant holder may fill which slot, are the
host's to decide: this module only records the slot labels and that each is
filled once by a different approver.
