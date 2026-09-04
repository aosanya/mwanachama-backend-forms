# mwanachama-backend-forms — documentation

## Layout

Four folders, in SDLC order, and everything lives under one of them.

| Folder | What's inside |
|--------|---------------|
| [1. requirements/](1.%20requirements/) | Problem, vision and scope for Form (survey) management extracted out of the gateway. |
| [2. design/](2.%20design/) | The Form/Question/Answer schema and how it maps onto GORM. |
| [3. implementation/](3.%20implementation/) | The work: `todo.md` (open board), `todo_done.md` (completed rows + board context). |
| [4. qa/](4.%20qa/) | Test coverage and results. |

## Boards and status

| File | What it holds |
| --- | --- |
| [todo.md](3.%20implementation/todo.md) | Open task board |
| [todo_done.md](3.%20implementation/todo_done.md) | Completed rows + board context |

## What this repo is

Extracts `mwanachama-backend-api-gateway`'s `survey` domain (renamed `Form`)
onto GORM-backed relational storage — domain logic and storage both live in
this package, imported directly by
[mwanachama-backend-api-gateway](../mwanachama-backend-api-gateway) — no
gRPC, no sub-service shape. Structured exactly like
[mwanachama-backend-actor](../mwanachama-backend-actor), the org's other
gateway-domain extraction. Built 2026-09-04, at the user's request; see this
repo's [CLAUDE.md](../CLAUDE.md) for the naming decision and every deviation
from a literal file-for-file copy of actor.
