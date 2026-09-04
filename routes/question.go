// question.go — HTTP routes over question_impl.go's Question/QuestionOption
// CRUD and versioning.
package routes

import (
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type optionBody struct {
	Label string `json:"label"`
}

func toOptions(bodies []optionBody) []models.QuestionOption {
	out := make([]models.QuestionOption, len(bodies))
	for i, b := range bodies {
		out[i] = models.QuestionOption{Label: b.Label}
	}
	return out
}

type questionBody struct {
	AnswerType   models.AnswerType `json:"answer_type"`
	Prompt       string            `json:"prompt"`
	Placeholder  string            `json:"placeholder,omitempty"`
	MaxLength    *int              `json:"max_length,omitempty"`
	UnitLabel    string            `json:"unit_label,omitempty"`
	Helper       string            `json:"helper,omitempty"`
	CurrencyCode string            `json:"currency_code,omitempty"`
	QuickPicks   []float64         `json:"quick_picks,omitempty"`
	YesLabel     string            `json:"yes_label,omitempty"`
	NoLabel      string            `json:"no_label,omitempty"`
	Options      []optionBody      `json:"options,omitempty"`
}

func (b questionBody) toQuestion() models.Question {
	return models.Question{
		AnswerType: b.AnswerType, Prompt: b.Prompt, Placeholder: b.Placeholder,
		MaxLength: b.MaxLength, UnitLabel: b.UnitLabel, Helper: b.Helper,
		CurrencyCode: b.CurrencyCode, QuickPicks: b.QuickPicks, YesLabel: b.YesLabel, NoLabel: b.NoLabel,
	}
}

type questionResponse struct {
	Question models.Question         `json:"question"`
	Options  []models.QuestionOption `json:"options"`
}

// AddQuestion handles POST {formID}/questions.
func AddQuestion(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body questionBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		q := body.toQuestion()
		q.FormID = r.PathValue("formID")
		question, options, err := fm.AddQuestion(r.Context(), q, toOptions(body.Options))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, questionResponse{Question: question, Options: options})
	}
}

// ListQuestions handles GET {formID}/questions — every version, Ordinal order.
func ListQuestions(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.ListQuestions(r.Context(), r.PathValue("formID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateQuestion handles PATCH {formID}/questions/{questionID} — draft-only,
// whole-question replace.
func UpdateQuestion(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body questionBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		question, options, err := fm.UpdateQuestion(r.Context(), r.PathValue("formID"), r.PathValue("questionID"), body.toQuestion(), toOptions(body.Options))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, questionResponse{Question: question, Options: options})
	}
}

// ListOptions handles GET {formID}/questions/{questionID}/options.
func ListOptions(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := fm.ListOptions(r.Context(), r.PathValue("questionID"))
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// AddOption handles POST {formID}/questions/{questionID}/options — the one
// additive content edit allowed regardless of the form's status.
func AddOption(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Label   string `json:"label"`
			ActorID string `json:"actor_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.AddOption(r.Context(), r.PathValue("questionID"), body.Label, body.ActorID)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// VersionQuestion handles POST {formID}/questions/{questionID}/versions —
// supersedes a published question with a new row.
func VersionQuestion(fm mwanachamaforms.FormManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			questionBody
			ActorID string `json:"actor_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := fm.VersionQuestion(r.Context(), r.PathValue("questionID"), body.toQuestion(), toOptions(body.Options), body.ActorID)
		if err != nil {
			writeFormErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}
