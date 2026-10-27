# A form that cannot be edited after it is sent

HS109, decision 41: a form that must not be edited after it is sent says so
when it is composed.

`Form.LockedAfterSend`, column and wire name `locked_after_send`, a plain
`bool`. It is named for the exception so that false, the zero value and the
default, means a member may resend until the form closes (HS90). Like
`window_advisory` it is chosen at `Create` and `Update` never touches it.

When it is set, `SubmitAnswers` looks for a submission by the same member,
group and period before writing. If one exists the call is refused with
`ErrLockedAfterSend`, declared 409 in `forms.operations.json`, and nothing is
written. A first send, another member's first send, and a new period of a
repeating form all still work.

The answered list carries the flag as `locked_after_send` so the app can
hide its edit button.
