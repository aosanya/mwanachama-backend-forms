package models

type TranslationSubject string

const (
	TranslationOfForm     TranslationSubject = "form"
	TranslationOfQuestion TranslationSubject = "question"
	TranslationOfOption   TranslationSubject = "question_option"
)

type Translation struct {
	ID          string             `json:"id"`
	FormID      string             `json:"form_id"`
	SubjectRole TranslationSubject `json:"subject_role"`
	SubjectID   string             `json:"subject_id"`
	Field       string             `json:"field"`
	Language    string             `json:"language"`
	Text        string             `json:"text"`
	Reviewed    bool               `json:"reviewed"`
	UpdatedAt   string             `json:"updated_at"`
}
