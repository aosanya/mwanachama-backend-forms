// target.go — HTTP routes over target_impl.go's Target CRUD.
package routes

import (
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// AddTarget handles POST {formID}/targets.
func AddTarget(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ChapterID           string `json:"chapter_id"`
			IncludesDescendants bool   `json:"includes_descendants"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.AddTarget(r.Context(), models.Target{
			FormID: r.PathValue("formID"), ChapterID: body.ChapterID, IncludesDescendants: body.IncludesDescendants,
		})
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// ListTargets handles GET {formID}/targets.
func ListTargets(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.ListTargets(r.Context(), r.PathValue("formID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RemoveTarget handles DELETE {formID}/targets/{targetID}.
func RemoveTarget(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fm.RemoveTarget(r.Context(), r.PathValue("formID"), r.PathValue("targetID")); err != nil {
			writeFormErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
