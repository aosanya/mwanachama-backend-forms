# Design & architecture board

New board — this repo isn't part of the mirrored 7-repo platform set
(`developer`, `mwanachama-frontend-*`, `mwanachama-network-admin*`,
`mwanachama-orchestrator`, `mwanachama-website*`) that shares a `DSN-XXX`
board verbatim, so it gets its own local numbering: **`FMD-XXX`**
(Forms Design), starting at FMD-001. Completed rows move to
`todo_done.md` (create it on first completion) — this file is pending-only.

Row format: `| ID | Pri | Status | Responsible | Phase | Task | Raised by | Depends On |`
— 📋 not started; Responsible is 🤖 agent (a routine works it), 👤 human (a
decision the user makes), or 🦸 super human (owner review).

| ID | Pri | Status | Responsible | Phase | Task | Raised by | Depends On |
|---|---|---|---|---|---|---|---|

FMD-001 is done — the decision record for the declared-domain conversion is
[todo_details/FMD-001.md](todo_details/FMD-001.md). The open design question
this repo still carries is not here but on
[../3. implementation/todo.md](../3.%20implementation/todo.md) as **F5**
(where collected data lands), because it is a missing capability to build
rather than an open question about how this repo is shaped.
