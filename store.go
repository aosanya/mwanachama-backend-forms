package mwanachamaforms

import (
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

// The roles this module operates on. A domain calls the objects whatever it
// likes — a survey, a census — and fills these roles with them; every rule
// in this package reaches its table through one of these and never through a
// domain's own noun.
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

// store is the spec-driven store, which lives in
// mwanachama-backend-shared/specstore: it knows which table plays which of
// the module's roles and how a domain value becomes a row, and nothing else.
// There are no row structs here, because the columns are declared.
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
