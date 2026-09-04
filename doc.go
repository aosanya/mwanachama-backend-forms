// Package mwanachamaforms provides Form lifecycle management — compose,
// approve, publish, answer, close — extracted from
// mwanachama-backend-api-gateway's internal/domain/survey package onto
// GORM-backed relational storage: domain logic AND storage both live in this
// package, imported directly into the gateway process — no separate
// service, no gRPC, no proto. Structured exactly like
// mwanachama-backend-actor (its extraction of internal/domain/member and
// internal/domain/chapter): same GORM-in-package pattern, same models/ +
// gormstore/ split, same route-builder shape. This repo's own
// naming-consistency pass renames the gateway's root noun Survey to Form —
// mirroring actor's own Member->Actor/Chapter->Group rename — including
// every SurveyID field, which becomes FormID.
//
// Layout:
//   - models/    — domain types (Form, Question, QuestionOption, Target,
//     Approval, Propagation, PublicLink, Respondent, Declaration, Answer)
//   - gormstore/ — GORM row structs, row<->domain conversion, migration
//   - doc.go (this file), tables.go — table-name/migrate wrappers
//   - manager.go            — FormManager interface, formManager struct
//   - form_impl.go          — Form CRUD and lifecycle (create/publish/close/delete)
//   - approval_impl.go      — the submit/withdraw/approve/refuse workflow
//   - question_impl.go      — Question/QuestionOption CRUD and versioning
//   - target_impl.go        — Target CRUD
//   - propagation_impl.go   — network rollup/pick-up tracking
//   - public_impl.go        — public link resolution, respondents, answers
//   - register_impl.go      — the search/list page
//   - errors.go             — sentinel errors
//
// Ported from mwanachama-backend-api-gateway's internal/domain/survey and
// internal/store/memory/survey_store*.go (the Go reference implementation —
// closer to this repo's own Go/GORM shape than the Postgres SQL store; see
// this repo's CLAUDE.md for what changed along the way, including what was
// deliberately not carried over).
package mwanachamaforms
