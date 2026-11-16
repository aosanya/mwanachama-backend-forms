package mwanachamaforms

import (
	"strings"
	"time"
)

var busyMessages = []string{
	"database table is locked",
	"database is locked",
	"database is deadlocked",
	"SQLITE_BUSY",
	"could not serialize access",
	"deadlock detected",
}

const busyBackoff = 4 * time.Millisecond

func isBusy(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	for _, busy := range busyMessages {
		if strings.Contains(message, busy) {
			return true
		}
	}
	return false
}
