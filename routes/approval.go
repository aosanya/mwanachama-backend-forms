// approval.go — HTTP routes over approval_impl.go's submit/withdraw/
// approve/refuse workflow (submit/withdraw live on form.go — they mutate
// the Form directly; this file covers the decision half).
package routes

import (
	"net/http"
	"strconv"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

type decisionBody struct {
	ApproverID string `json:"approver_id"`
	Note       string `json:"note,omitempty"`
}

// ApproveForm handles POST {formID}/approvals/approve.
func ApproveForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body decisionBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Approve(r.Context(), r.PathValue("formID"), body.ApproverID, body.Note)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RefuseForm handles POST {formID}/approvals/refuse.
func RefuseForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body decisionBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Refuse(r.Context(), r.PathValue("formID"), body.ApproverID, body.Note)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListApprovals handles GET {formID}/approvals — a form's decision history,
// newest first.
func ListApprovals(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.ListApprovals(r.Context(), r.PathValue("formID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListAwaitingApproval handles GET approvals/awaiting — every submitted
// form network-wide, SubmittedAt order. Query param: limit (0 or absent =
// unlimited).
func ListAwaitingApproval(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				writeErr(w, http.StatusBadRequest, "limit: "+err.Error())
				return
			}
			limit = n
		}
		out, err := fm.ListAwaitingApproval(r.Context(), limit)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
