package mwanachamaforms

import "errors"

var (
	ErrGrantOnRefusal    = errors.New("mwanachamaforms: a certificate and an end day belong to an approval, not a refusal")
	ErrInvalidValidUntil = errors.New("mwanachamaforms: the end day is a day written YYYY-MM-DD")
	ErrValidUntilPast    = errors.New("mwanachamaforms: the end day cannot be before today")
)
