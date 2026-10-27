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

## A form table that predates the column

No dedicated step is needed. `Provision` runs `addMissingColumns` after
`spec.Migrate`, which adds every declared column an existing table lacks, on
sqlite and postgres alike, and skips one already present, so a second run
changes nothing. `locked_after_send` is an optional bool, so it is added as
`boolean default false`: forms made before the field read back as editable.
`hs109_provision_test.go` builds the old shape by dropping the column, runs
`Provision` twice and checks the column, an earlier form reading unlocked and
a new locked form; it fails when `addMissingColumns` is removed.
