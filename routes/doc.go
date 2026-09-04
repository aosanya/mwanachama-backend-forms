// Package routes is mwanachama-backend-forms's own HTTP surface: decode a
// request, call one [mwanachamaforms.FormManager] method, encode the
// response — structured exactly like mwanachama-backend-actor/routes,
// including its central invariant: a route built here needs a caller-
// identity/capability gate wrapped around it before it is safe to serve.
// This package answers "what happens once that gate has passed", never "who
// may pass it" — no session, no Role/capability check, no chapter-descendant
// validation of a Target's ChapterID.
//
// Scope, as of this repo's first build: [FormRoutes] (create/list-search/
// get/update/delete/submit/withdraw/publish/close), [ApprovalRoutes]
// (approve/refuse/list/list-awaiting), [QuestionRoutes] (add/list/update/
// options/add-option/version), [TargetRoutes] (add/list/remove),
// [PropagationRoutes] (list/rollup/pick-up), [AnswerRoutes] (submit/list),
// and [PublicLinkRoutes] (resolve/upsert-respondent/declare). [Routes]
// returns the whole set.
//
// Deliberately not wired to HTTP here, and not a future TODO — a considered
// exclusion, the same shape as mwanachama-backend-actor/routes' doc.go:
// FormManager.ListForChapter and .ListReachable are both plain single-call
// operations, but [FormRoutes]' GET (Register) already covers the same
// chapter-scoped listing need with richer counts attached, so exposing a
// second, narrower listing endpoint for the same use case was skipped to
// keep this initial route surface's size in proportion to the rest of this
// repo, not because either method depends on anything this package could
// not import.
package routes
