# Requirements

Form (survey) management for the Mwanachama network — composing a form,
routing it through approval, publishing it to a chapter's descendants or the
public, collecting answers, and closing it. Extracted from
`mwanachama-backend-api-gateway`'s `internal/domain/survey` package onto
GORM, structured exactly like `mwanachama-backend-actor`'s own extraction of
the gateway's `member`/`chapter` domains. See this repo's `CLAUDE.md` for the
naming decision (`Survey` → `Form`) and every deviation from a literal
file-for-file copy of actor.

The underlying functional requirements — compose/publish, answer, public
link, reminders, localization, interviews — are the same ones already
written up in the product's own story-board docs (`todo_survey_*.md`
wherever those are mirrored across repos); this repo does not restate them,
and does not implement the pieces the gateway's own domain never implemented
either (reminders, localization, cascading push-down — see CLAUDE.md's
"inherited dead vocabulary" note).
