package mwanachamaforms

import (
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

const (
	roleForm           = "form"
	roleQuestion       = "question"
	roleQuestionOption = "question_option"
	roleTarget         = "target"
	roleApproval       = "approval"
	rolePropagation    = "propagation"
	rolePublicLink     = "public_link"
	roleRespondent     = "respondent"
	roleDeclaration    = "declaration"
	roleAnswer         = "answer"
)

type store = specstore.Store

func newStore(db *gorm.DB, s *spec.Spec, carriers map[string]any) (*store, error) {
	return specstore.New(db, s, carriers)
}

func newID() string { return specstore.NewID() }

func columnName(field string) string { return specstore.ColumnName(field) }

func encode(o spec.Object, v any) (map[string]any, error) { return specstore.Encode(o, v) }

func decode(o spec.Object, row map[string]any, out any) error {
	return specstore.Decode(o, row, out)
}
