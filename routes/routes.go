package routes

import (
	"net/http"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

// Route is one address this package answers, relative to wherever the
// mounting process prefixes it. See mwanachama-backend-actor/routes.Route,
// which this mirrors exactly.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Pattern returns the http.ServeMux registration pattern for this route
// once mounted under prefix.
func (r Route) Pattern(prefix string) string {
	return r.Method + " " + prefix + r.Path
}

// ResourceNames configures the URL noun each type is addressed under, every
// field defaulting to this package's own noun when left "" — the same
// org-configurable-label shape mwanachama-backend-actor/routes.ResourceNames
// provides for Group.
type ResourceNames struct {
	Form        string
	Question    string
	Target      string
	Approval    string
	Propagation string
	PublicLink  string
	Answer      string
}

func (n ResourceNames) withDefaults() ResourceNames {
	if n.Form == "" {
		n.Form = "forms"
	}
	if n.Question == "" {
		n.Question = "questions"
	}
	if n.Target == "" {
		n.Target = "targets"
	}
	if n.Approval == "" {
		n.Approval = "approvals"
	}
	if n.Propagation == "" {
		n.Propagation = "propagation"
	}
	if n.PublicLink == "" {
		n.PublicLink = "public-links"
	}
	if n.Answer == "" {
		n.Answer = "answers"
	}
	return n
}

// FormRoutes is Form's create/list-search/get/update/delete plus its
// lifecycle actions (submit/withdraw/publish/close), addressed under
// names.Form (default "forms").
func FormRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Form
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateForm(fm)},
		{Method: http.MethodGet, Path: base, Handler: RegisterForms(fm)},
		{Method: http.MethodGet, Path: base + "/{formID}", Handler: GetForm(fm)},
		{Method: http.MethodPatch, Path: base + "/{formID}", Handler: UpdateForm(fm)},
		{Method: http.MethodDelete, Path: base + "/{formID}", Handler: DeleteForm(fm)},
		{Method: http.MethodPost, Path: base + "/{formID}/submit", Handler: SubmitForm(fm)},
		{Method: http.MethodPost, Path: base + "/{formID}/withdraw", Handler: WithdrawForm(fm)},
		{Method: http.MethodPost, Path: base + "/{formID}/publish", Handler: PublishForm(fm)},
		{Method: http.MethodPost, Path: base + "/{formID}/close", Handler: CloseForm(fm)},
	}
}

// ApprovalRoutes is the submitted-form decision workflow: approve/refuse a
// form's open approval, list a form's approval history, and list every form
// awaiting a decision network-wide.
func ApprovalRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	formBase := "/" + names.Form + "/{formID}/" + names.Approval
	return []Route{
		{Method: http.MethodPost, Path: formBase + "/approve", Handler: ApproveForm(fm)},
		{Method: http.MethodPost, Path: formBase + "/refuse", Handler: RefuseForm(fm)},
		{Method: http.MethodGet, Path: formBase, Handler: ListApprovals(fm)},
		{Method: http.MethodGet, Path: "/" + names.Approval + "/awaiting", Handler: ListAwaitingApproval(fm)},
	}
}

// QuestionRoutes is Question/QuestionOption CRUD plus versioning, nested
// under a form.
func QuestionRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Form + "/{formID}/" + names.Question
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: AddQuestion(fm)},
		{Method: http.MethodGet, Path: base, Handler: ListQuestions(fm)},
		{Method: http.MethodPatch, Path: base + "/{questionID}", Handler: UpdateQuestion(fm)},
		{Method: http.MethodGet, Path: base + "/{questionID}/options", Handler: ListOptions(fm)},
		{Method: http.MethodPost, Path: base + "/{questionID}/options", Handler: AddOption(fm)},
		{Method: http.MethodPost, Path: base + "/{questionID}/versions", Handler: VersionQuestion(fm)},
	}
}

// TargetRoutes is Target CRUD, nested under a form.
func TargetRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Form + "/{formID}/" + names.Target
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: AddTarget(fm)},
		{Method: http.MethodGet, Path: base, Handler: ListTargets(fm)},
		{Method: http.MethodDelete, Path: base + "/{targetID}", Handler: RemoveTarget(fm)},
	}
}

// PropagationRoutes is the network-rollup/pick-up surface, nested under a
// form.
func PropagationRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Form + "/{formID}/" + names.Propagation
	return []Route{
		{Method: http.MethodGet, Path: base, Handler: ListPropagation(fm)},
		{Method: http.MethodGet, Path: base + "/rollup", Handler: RollupForm(fm)},
		{Method: http.MethodPost, Path: base + "/pick-up", Handler: PickUpForm(fm)},
	}
}

// AnswerRoutes is Answer submit/list, nested under a form. Caller-identity
// (who "member" is) is entirely the mounting process's concern — this
// package reads it from the request body/path, never a session.
func AnswerRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Form + "/{formID}/" + names.Answer
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: SubmitAnswers(fm)},
		{Method: http.MethodGet, Path: base + "/{memberID}", Handler: ListAnswers(fm)},
	}
}

// PublicLinkRoutes is the anonymous, no-account entry point for a
// public-audience form: resolve a link key, upsert a respondent, and
// declare a respondent's chapter.
func PublicLinkRoutes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.PublicLink + "/{key}"
	return []Route{
		{Method: http.MethodGet, Path: base, Handler: ResolveLinkKey(fm)},
		{Method: http.MethodPost, Path: base + "/respondents", Handler: UpsertRespondent(fm)},
		{Method: http.MethodPost, Path: base + "/declarations", Handler: DeclareChapter(fm)},
	}
}

// Routes is every address this package answers today, sharing one
// ResourceNames.
func Routes(fm mwanachamaforms.FormManager, names ResourceNames) []Route {
	out := FormRoutes(fm, names)
	out = append(out, ApprovalRoutes(fm, names)...)
	out = append(out, QuestionRoutes(fm, names)...)
	out = append(out, TargetRoutes(fm, names)...)
	out = append(out, PropagationRoutes(fm, names)...)
	out = append(out, AnswerRoutes(fm, names)...)
	out = append(out, PublicLinkRoutes(fm, names)...)
	return out
}
