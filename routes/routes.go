package routes

import (
	"fmt"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

type Route = httpwire.Route

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(forms.Operations())
})

var sentinels = map[string]error{
	"ErrFormNotFound":        forms.ErrFormNotFound,
	"ErrQuestionNotFound":    forms.ErrQuestionNotFound,
	"ErrTargetNotFound":      forms.ErrTargetNotFound,
	"ErrLinkNotFound":        forms.ErrLinkNotFound,
	"ErrNotDraft":            forms.ErrNotDraft,
	"ErrNotWithdrawable":     forms.ErrNotWithdrawable,
	"ErrNotSubmitted":        forms.ErrNotSubmitted,
	"ErrAwaitingApproval":    forms.ErrAwaitingApproval,
	"ErrNotOpen":             forms.ErrNotOpen,
	"ErrPublished":           forms.ErrPublished,
	"ErrPublicFormNoTargets": forms.ErrPublicFormNoTargets,
	"ErrFormNotPublished":    forms.ErrFormNotPublished,
	"ErrFormNotOpen":         forms.ErrFormNotOpen,
	"ErrNotCurrentVersion":   forms.ErrNotCurrentVersion,
	"ErrDuplicateTarget":     forms.ErrDuplicateTarget,
	"ErrLinkKeyTaken":        forms.ErrLinkKeyTaken,
	"ErrInvalidReference":    forms.ErrInvalidReference,
	"ErrInvalidForm":         forms.ErrInvalidForm,
	"ErrInvalidQuestion":     forms.ErrInvalidQuestion,
	"ErrInvalidAnswer":       forms.ErrInvalidAnswer,
	"ErrMissingTitle":        forms.ErrMissingTitle,
	"ErrMissingClosesAt":     forms.ErrMissingClosesAt,
	"ErrClosesInPast":        forms.ErrClosesInPast,
	"ErrWindowInvalid":       forms.ErrWindowInvalid,
	"ErrAudienceConflict":    forms.ErrAudienceConflict,
	"ErrMissingPrompt":       forms.ErrMissingPrompt,
	"ErrMissingOptionLabel":  forms.ErrMissingOptionLabel,
	"ErrMissingNote":         forms.ErrMissingNote,
	"ErrQuestionNotOnForm":   forms.ErrQuestionNotOnForm,
}

var AnonymousActions = []string{
	"forms.public_link.open",
	"forms.respondent.register",
	"forms.declaration.create",
}

type Mount struct {
	Authorize dispatch.Authorizer
	Caller    dispatch.Caller
}

func Build(fm forms.FormManager) ([]Route, error) { return BuildFor(fm, Mount{}) }

func BuildFor(fm forms.FormManager, m Mount) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{
		Manager: fm, Errors: sentinels, Authorize: m.Authorize, Caller: m.Caller,
	})
}

func Routes(fm forms.FormManager) []Route { return RoutesFor(fm, Mount{}) }

func RoutesFor(fm forms.FormManager, m Mount) []Route {
	out, err := BuildFor(fm, m)
	if err != nil {
		panic(fmt.Sprintf("forms routes: %v", err))
	}
	return out
}

func Split(fm forms.FormManager) dispatch.Split { return SplitFor(fm, Mount{}) }

func SplitFor(fm forms.FormManager, m Mount) dispatch.Split {
	public := dispatch.Anonymous(Routes(fm), AnonymousActions...)
	gated := dispatch.Anonymous(RoutesFor(fm, m), AnonymousActions...)
	return dispatch.Split{Anonymous: public.Anonymous, Gated: gated.Gated}
}

func PublicRoutes(fm forms.FormManager) []Route { return Split(fm).Anonymous }

func OperatorRoutes(fm forms.FormManager) []Route { return OperatorRoutesFor(fm, Mount{}) }

func OperatorRoutesFor(fm forms.FormManager, m Mount) []Route { return SplitFor(fm, m).Gated }
