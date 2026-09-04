// propagation.go — HTTP routes over propagation_impl.go's network rollup/
// pick-up tracking.
package routes

import (
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

// ListPropagation handles GET {formID}/propagation.
func ListPropagation(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.ListPropagation(r.Context(), r.PathValue("formID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RollupForm handles GET {formID}/propagation/rollup.
func RollupForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.Rollup(r.Context(), r.PathValue("formID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// PickUpForm handles POST {formID}/propagation/pick-up.
func PickUpForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ChapterID string `json:"chapter_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.PickUp(r.Context(), r.PathValue("formID"), body.ChapterID)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
