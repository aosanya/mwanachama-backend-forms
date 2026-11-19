package mwanachamaforms

import "errors"

var (
	ErrInvalidInspection = errors.New("mwanachamaforms: a finding on a send is complies, does_not_comply or needs_more, by a named person")
	ErrSelfInspection    = errors.New("mwanachamaforms: an applicant cannot inspect their own send")
)
