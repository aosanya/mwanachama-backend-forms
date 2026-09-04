package routes

import (
	"encoding/json"
	"errors"
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

// writeJSON and writeErr mirror the gateway's own internal/api/http/wire.go
// byte-for-byte on purpose — see mwanachama-backend-actor/routes/wire.go,
// which this is copied from verbatim.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// formStatusFor maps this package's error sentinels to a status code. Unlike
// mwanachama-backend-actor's per-resource statusFor functions, this domain's
// errors are shared across every handler file (ErrNotDraft alone guards
// Update, AddQuestion, UpdateQuestion, AddTarget, RemoveTarget), so one
// function keeps the mapping consistent rather than risking three files
// disagreeing on the same sentinel's status.
func formStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaforms.ErrFormNotFound),
		errors.Is(err, mwanachamaforms.ErrQuestionNotFound),
		errors.Is(err, mwanachamaforms.ErrTargetNotFound):
		return http.StatusNotFound

	case errors.Is(err, mwanachamaforms.ErrNotDraft),
		errors.Is(err, mwanachamaforms.ErrNotWithdrawable),
		errors.Is(err, mwanachamaforms.ErrNotSubmitted),
		errors.Is(err, mwanachamaforms.ErrAwaitingApproval),
		errors.Is(err, mwanachamaforms.ErrNotOpen),
		errors.Is(err, mwanachamaforms.ErrPublished),
		errors.Is(err, mwanachamaforms.ErrPublicFormNoTargets),
		errors.Is(err, mwanachamaforms.ErrFormNotPublished),
		errors.Is(err, mwanachamaforms.ErrFormNotOpen),
		errors.Is(err, mwanachamaforms.ErrNotCurrentVersion),
		errors.Is(err, mwanachamaforms.ErrDuplicateTarget):
		return http.StatusConflict

	case errors.Is(err, mwanachamaforms.ErrInvalidReference),
		errors.Is(err, mwanachamaforms.ErrMissingTitle),
		errors.Is(err, mwanachamaforms.ErrMissingClosesAt),
		errors.Is(err, mwanachamaforms.ErrClosesInPast),
		errors.Is(err, mwanachamaforms.ErrWindowInvalid),
		errors.Is(err, mwanachamaforms.ErrAudienceConflict),
		errors.Is(err, mwanachamaforms.ErrMissingPrompt),
		errors.Is(err, mwanachamaforms.ErrMissingOptionLabel),
		errors.Is(err, mwanachamaforms.ErrMissingNote),
		errors.Is(err, mwanachamaforms.ErrInvalidAnswer),
		errors.Is(err, mwanachamaforms.ErrQuestionNotOnForm):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

func writeFormErr(w http.ResponseWriter, err error) {
	code := formStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}
