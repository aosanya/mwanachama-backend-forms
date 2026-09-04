// form.go — HTTP routes over form_impl.go's Create/Get/Update/Delete/
// Register and lifecycle actions.
package routes

import (
	"net/http"
	"strconv"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// CreateForm handles POST — decode, create, encode.
func CreateForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Form
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Create(r.Context(), in)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// RegisterForms handles GET — a filtered, paginated search page. Query
// params: q (title substring), chapter_id, status, limit, offset.
func RegisterForms(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := models.RegisterQuery{
			Search:    r.URL.Query().Get("q"),
			ChapterID: r.URL.Query().Get("chapter_id"),
			Status:    models.Status(r.URL.Query().Get("status")),
		}
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				writeErr(w, http.StatusBadRequest, "limit: "+err.Error())
				return
			}
			q.Limit = &n
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				writeErr(w, http.StatusBadRequest, "offset: "+err.Error())
				return
			}
			q.Offset = n
		}
		out, err := fm.Register(r.Context(), q)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// GetForm handles GET {formID}.
func GetForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.Get(r.Context(), r.PathValue("formID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// formEditBody is the wire shape for UpdateForm — the five columns
// UserManager.Update writes.
type formEditBody struct {
	Title          string                `json:"title"`
	ClosesAt       string                `json:"closes_at"`
	OpensAt        string                `json:"opens_at"`
	Audience       models.Audience       `json:"audience"`
	CollectionMode models.CollectionMode `json:"collection_mode"`
}

// UpdateForm handles PATCH {formID}.
func UpdateForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body formEditBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Update(r.Context(), models.Form{
			ID: r.PathValue("formID"), Title: body.Title, ClosesAt: body.ClosesAt,
			OpensAt: body.OpensAt, Audience: body.Audience, CollectionMode: body.CollectionMode,
		})
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteForm handles DELETE {formID}.
func DeleteForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fm.Delete(r.Context(), r.PathValue("formID")); err != nil {
			writeFormErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// SubmitForm handles POST {formID}/submit.
func SubmitForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SubmittedBy string `json:"submitted_by"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Submit(r.Context(), r.PathValue("formID"), body.SubmittedBy)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// WithdrawForm handles POST {formID}/withdraw.
func WithdrawForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ActorID string `json:"actor_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Withdraw(r.Context(), r.PathValue("formID"), body.ActorID)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// publishResponse carries both of Publish's return values — a draft/
// approved member-audience form's response has an empty "public_link".
type publishResponse struct {
	Form       models.Form       `json:"form"`
	PublicLink models.PublicLink `json:"public_link,omitzero"`
}

// PublishForm handles POST {formID}/publish.
func PublishForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CandidateLinkKey string `json:"candidate_link_key"`
			PublishedBy      string `json:"published_by"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		f, link, err := fm.Publish(r.Context(), r.PathValue("formID"), body.CandidateLinkKey, body.PublishedBy)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, publishResponse{Form: f, PublicLink: link})
	}
}

// CloseForm handles POST {formID}/close.
func CloseForm(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ClosedBy string `json:"closed_by"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Close(r.Context(), r.PathValue("formID"), body.ClosedBy)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
