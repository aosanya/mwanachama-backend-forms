// answer.go — HTTP routes over public_impl.go's SubmitAnswers/ListAnswers.
// No caller-identity extraction here — member_id travels in the request
// body/path, the mounting process's own concern (see doc.go).
package routes

import (
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

type answerBody struct {
	QuestionID  string   `json:"question_id"`
	OptionIDs   []string `json:"option_ids,omitempty"`
	ValueText   string   `json:"value_text,omitempty"`
	ValueNumber *float64 `json:"value_number,omitempty"`
	ValueDate   string   `json:"value_date,omitempty"`
	ValueTime   string   `json:"value_time,omitempty"`
	ValueBool   *bool    `json:"value_bool,omitempty"`
}

// SubmitAnswers handles POST {formID}/answers — one all-or-nothing batch.
func SubmitAnswers(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			MemberID  string       `json:"member_id"`
			ChapterID string       `json:"chapter_id"`
			Answers   []answerBody `json:"answers"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		answers := make([]mwanachamaforms.Answer, len(body.Answers))
		for i, a := range body.Answers {
			answers[i] = mwanachamaforms.Answer{
				QuestionID: a.QuestionID, OptionIDs: a.OptionIDs, ValueText: a.ValueText,
				ValueNumber: a.ValueNumber, ValueDate: a.ValueDate, ValueTime: a.ValueTime, ValueBool: a.ValueBool,
			}
		}
		out, err := fm.SubmitAnswers(r.Context(), r.PathValue("formID"), body.MemberID, body.ChapterID, answers)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListAnswers handles GET {formID}/answers/{memberID}.
func ListAnswers(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.ListAnswers(r.Context(), r.PathValue("formID"), r.PathValue("memberID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
