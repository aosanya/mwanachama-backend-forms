package routes_test

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func TestTheAddressTableIsWhatTheSpecDeclares(t *testing.T) {
	got := make([]string, 0)
	for _, rt := range routes.Routes(newTestManager(t)) {
		got = append(got, rt.Method+" "+rt.Path)
	}
	sort.Strings(got)

	want := []string{
		"DELETE /forms/{form_id}",
		"DELETE /forms/{form_id}/targets/{target_id}",
		"GET /approvals/awaiting",
		"GET /forms",
		"GET /forms/{form_id}",
		"GET /forms/{form_id}/answers/{member_id}",
		"GET /forms/{form_id}/approvals",
		"GET /forms/{form_id}/propagation",
		"GET /forms/{form_id}/propagation/rollup",
		"GET /forms/{form_id}/questions",
		"GET /forms/{form_id}/questions/{question_id}/options",
		"GET /forms/{form_id}/targets",
		"GET /public-links/{key}",
		"PATCH /forms/{form_id}",
		"PATCH /forms/{form_id}/questions/{question_id}",
		"POST /forms",
		"POST /forms/{form_id}/answers",
		"POST /forms/{form_id}/approvals/approve",
		"POST /forms/{form_id}/approvals/refuse",
		"POST /forms/{form_id}/close",
		"POST /forms/{form_id}/propagation/pick-up",
		"POST /forms/{form_id}/publish",
		"POST /forms/{form_id}/questions",
		"POST /forms/{form_id}/questions/{question_id}/options",
		"POST /forms/{form_id}/questions/{question_id}/versions",
		"POST /forms/{form_id}/submit",
		"POST /forms/{form_id}/targets",
		"POST /forms/{form_id}/withdraw",
		"POST /public-links/{key}/declarations",
		"POST /public-links/{key}/respondents",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("got %d addresses, want %d:\n got: %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("address %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEveryAddressArrivesGated(t *testing.T) {
	for _, rt := range routes.Routes(newTestManager(t)) {
		if rt.Action == "" {
			t.Errorf("%s %s carries no action", rt.Method, rt.Path)
		}
	}
}

func TestOnlyTheLinkSurfaceIsAnonymous(t *testing.T) {
	got := make([]string, 0)
	for _, rt := range routes.PublicRoutes(newTestManager(t)) {
		got = append(got, rt.Action)
	}
	sort.Strings(got)

	want := []string{"forms.declaration.create", "forms.public_link.open", "forms.respondent.register"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("anonymous actions = %v, want %v", got, want)
	}
}

func TestRoute_PatternWithPrefix(t *testing.T) {
	for _, rt := range routes.Routes(newTestManager(t)) {
		if rt.Method != http.MethodPost || rt.Path != "/forms" {
			continue
		}
		if got := rt.Pattern("/v1"); got != "POST /v1/forms" {
			t.Fatalf("Pattern = %q", got)
		}
		return
	}
	t.Fatal("POST /forms is not in the table")
}
