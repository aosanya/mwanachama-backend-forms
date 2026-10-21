# MCP tools

`routes.BuildGatedTools(fm, mount)` builds one MCP tool per declared operation
in `forms.operations.json`, through `mwanachama-backend-shared/dispatch.Tools`,
from the same table the REST routes come from (HS47 on the Hub Steward board).

- **Only the gated half.** The four `AnonymousActions` (open a public link,
  read its questions, register a respondent, declare a group) never become
  tools. A tool always runs for a caller, so a signed-out public-link surface
  has no MCP counterpart.
- **The mount is the REST mount.** `Authorize` and `Caller` pass straight into
  `dispatch`, so every `from: caller` argument (`member_id`, `actor_id`,
  `created_by`, `published_by`, a claim's member) is bound from the session
  and stays out of the tool's input schema.
- **`ReachedOnly` narrows a tool exactly as it narrows a route**: the call's
  context carries `forms.OnlyReached` for the caller before the manager is
  reached, so a resident sees only the published forms that reached one of
  their groups.

The host adds its own instance argument and membership check around these
tools; this package knows nothing about agencies.
