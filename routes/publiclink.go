// publiclink.go — HTTP routes over public_impl.go's anonymous, no-account
// entry point: resolve a link key, upsert a respondent, declare a
// respondent's chapter. Every failure here is the same indistinguishable
// 404 ResolveLinkKey itself returns (found=false, never an error) — a
// public caller cannot tell a wrong key from a retired link from a closed
// form.
package routes

import (
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func writeLinkNotFound(w http.ResponseWriter) {
	writeErr(w, http.StatusNotFound, "not found")
}

type resolvedLinkResponse struct {
	Link mwanachamaforms.PublicLink `json:"link"`
	Form mwanachamaforms.Form       `json:"form"`
}

// ResolveLinkKey handles GET {key}.
func ResolveLinkKey(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		link, f, found, err := fm.ResolveLinkKey(r.Context(), r.PathValue("key"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		if !found {
			writeLinkNotFound(w)
			return
		}
		writeJSON(w, http.StatusOK, resolvedLinkResponse{Link: link, Form: f})
	}
}

// UpsertRespondent handles POST {key}/respondents. The key must still
// resolve — an unknown/retired/closed key refuses the same as ResolveLinkKey.
func UpsertRespondent(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, found, err := fm.ResolveLinkKey(r.Context(), r.PathValue("key"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		if !found {
			writeLinkNotFound(w)
			return
		}
		var body struct {
			PublicKey string `json:"public_key"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.UpsertRespondent(r.Context(), mwanachamaforms.Respondent{PublicKey: body.PublicKey})
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out.PublicView())
	}
}

// DeclareChapter handles POST {key}/declarations.
func DeclareChapter(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, f, found, err := fm.ResolveLinkKey(r.Context(), r.PathValue("key"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		if !found {
			writeLinkNotFound(w)
			return
		}
		var body struct {
			RespondentID      string `json:"respondent_id"`
			DeclaredChapterID string `json:"declared_chapter_id,omitempty"`
			DeclaredText      string `json:"declared_text,omitempty"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.Declare(r.Context(), mwanachamaforms.Declaration{
			RespondentID: body.RespondentID, FormID: f.ID,
			DeclaredChapterID: body.DeclaredChapterID, DeclaredText: body.DeclaredText,
		})
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}
