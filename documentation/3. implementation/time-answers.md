# Time answers

A time answer carries `value_time` as `HH:MM`, two digits for the hour and two for the minute, `00:00` to `23:59`. Any other spelling is refused with 400 and `time answer requires value_time as HH:MM`: `7:05`, `07:5`, `24:00`, `23:60`, `07:05:00`, a leading space, an empty value.

## Why

A date answer already has to match `2006-01-02` exactly, so `2026-1-5` is refused and is never rewritten. A time answer follows the same rule instead of being normalised: the text a person typed is the text stored, and two stored values compare and sort as text in one answer column. Go's `15` layout reads a one-digit hour, so `time.Parse("15:04", "7:05")` succeeds; `checkClockTime` in `models/validators.go` parses with `ClockLayout` and then requires the parsed time, written back, to equal the typed text.

## Rows stored before the rule

Answers already stored with a one-digit hour are left as they are. Rewriting them to `07:05` is an owner decision (hub issue MUN700, Blocked todo TD-1830).

## Tests

`models/answer_validators_cover_test.go` holds the accepted and refused spellings; `routes/time_answer_strict_test.go` sends `7:05` and `07:05` through the route table and reads the answers back.

## Hub copy

The hub copy (`projects/hub-steward/code/wakala-api/modules/forms/models`) has the one-digit hour accepted until the same change is replayed there or hub issue ISS-105 removes the copy. Its skipped probe `TestPollFormDates_FormTimeAnswerWithAOneDigitHourIsNotStoredAsTyped` turns green with the change and its `t.Skip` goes.
