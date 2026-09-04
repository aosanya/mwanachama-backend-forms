package models

import "time"

// TimeLayout is the fixed-width, nanosecond-precision RFC 3339 layout every
// timestamp field on a model in this package is written and read with.
//
// Plain time.RFC3339 (second precision) is not enough: two rows written
// microseconds apart in the same second would compare equal and any sort
// keyed on the field would fall through to a tie-break that does not track
// real order. time.RFC3339Nano is not safe either: Go trims trailing
// fractional zeros, so two timestamps with different digit counts stop
// comparing correctly as plain strings — see
// mwanachama-backend-actor/models/time.go, which this is copied from
// verbatim, for the parity test that caught exactly this. A fixed nine-digit
// fractional part keeps every stored timestamp both parseable AND
// lexicographically sortable in the same order as chronologically.
const TimeLayout = "2006-01-02T15:04:05.000000000Z07:00"

// NowRFC3339 returns the current UTC time formatted per [TimeLayout].
func NowRFC3339() string {
	return time.Now().UTC().Format(TimeLayout)
}
