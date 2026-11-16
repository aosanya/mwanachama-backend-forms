package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type closing struct {
	forms.FormManager
	err      error
	closedBy string
	calls    int
}

func (c *closing) Close(_ context.Context, id, by string) (models.Form, error) {
	c.calls++
	c.closedBy = by
	return models.Form{ID: id}, c.err
}

func closeRoute(t *testing.T, routes []Route) Route {
	t.Helper()
	for _, route := range routes {
		if route.Action == "forms.form.close" {
			return route
		}
	}
	t.Fatal("no forms.form.close route")
	return Route{}
}

func closeStatus(t *testing.T, manager forms.FormManager, mount Mount) *httptest.ResponseRecorder {
	t.Helper()
	routes, err := BuildFor(manager, mount)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req.SetPathValue("form_id", "f1")
	rec := httptest.NewRecorder()
	closeRoute(t, routes).Handler(rec, req)
	return rec
}

func declaredStatuses(t *testing.T) map[string]int {
	t.Helper()
	raw, err := os.ReadFile("../forms.operations.json")
	if err != nil {
		t.Fatalf("read operations: %v", err)
	}
	var file struct {
		Errors map[string]int `json:"errors"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse operations: %v", err)
	}
	return file.Errors
}

func TestEverySentinelTheFormsRoutesNameIsDeclaredWithAStatusAndTheOtherWayRound(t *testing.T) {
	declared := declaredStatuses(t)
	for name := range sentinels {
		if _, ok := declared[name]; !ok {
			t.Errorf("sentinel %s has no declared status", name)
		}
	}
	for name := range declared {
		if _, ok := sentinels[name]; !ok {
			t.Errorf("declared status for %s names no sentinel", name)
		}
	}
}

func TestAManagerRefusalAnswersTheStatusItsSentinelDeclares(t *testing.T) {
	declared := declaredStatuses(t)
	for name, want := range declared {
		sentinel, ok := sentinels[name]
		if !ok {
			continue
		}
		rec := closeStatus(t, &closing{err: sentinel}, sessionMount())
		if rec.Code != want {
			t.Errorf("%s answered %d %s, want %d", name, rec.Code, rec.Body.String(), want)
		}
	}
}

func TestAWrappedSentinelStillAnswersItsDeclaredStatus(t *testing.T) {
	rec := closeStatus(t, &closing{err: fmt.Errorf("closing f1: %w", forms.ErrFormNotFound)}, sessionMount())
	if rec.Code != http.StatusNotFound {
		t.Errorf("wrapped ErrFormNotFound answered %d %s, want 404", rec.Code, rec.Body.String())
	}
}

func TestAnUnlistedFailureIsNeitherAClientErrorNorLeaksItsText(t *testing.T) {
	rec := closeStatus(t, &closing{err: errors.New("pq: password for user steward rejected")}, sessionMount())
	if rec.Code < 500 {
		t.Errorf("an unexpected failure answered %d, want a 5xx", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("internal failure text reached the caller: %s", rec.Body.String())
	}
}

func TestAClosedFormIsAnswered200AndRecordsTheCaller(t *testing.T) {
	manager := &closing{}
	rec := closeStatus(t, manager, sessionMount())
	if rec.Code != http.StatusOK {
		t.Fatalf("close answered %d %s, want 200", rec.Code, rec.Body.String())
	}
	if manager.closedBy != "the-session" {
		t.Errorf("closed by %q, want the-session", manager.closedBy)
	}
	var body models.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.ID != "f1" {
		t.Errorf("body %s decoded to %+v (%v), want form f1", rec.Body.String(), body, err)
	}
}

func TestARefusingAuthorizerStopsEveryGatedFormsRouteBeforeTheManagerIsReached(t *testing.T) {
	asked := map[string]bool{}
	mount := Mount{
		Caller: func(context.Context) string { return "the-session" },
		Authorize: func(_ context.Context, action string) error {
			asked[action] = true
			return errors.New("no")
		},
	}
	manager := &closing{}
	gated := OperatorRoutesFor(manager, mount)
	if len(gated) == 0 {
		t.Fatal("no gated routes")
	}
	for _, route := range gated {
		req := httptest.NewRequest(route.Method, "/", strings.NewReader(`{}`))
		req.SetPathValue("form_id", "f1")
		rec := httptest.NewRecorder()
		route.Handler(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s answered %d, want 403", route.Action, rec.Code)
		}
		if !asked[route.Action] {
			t.Errorf("authorizer was never asked about %s", route.Action)
		}
	}
	if manager.calls != 0 {
		t.Errorf("manager reached %d times behind a refusing authorizer", manager.calls)
	}
}

func TestSplitSeparatesTheFourAnonymousActionsFromEveryGatedRoute(t *testing.T) {
	built, err := Build(&closing{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	split := Split(&closing{})
	if len(split.Anonymous)+len(split.Gated) != len(built) {
		t.Errorf("anonymous %d + gated %d, want %d routes in all", len(split.Anonymous), len(split.Gated), len(built))
	}
	for _, route := range split.Anonymous {
		if !slices.Contains(AnonymousActions, route.Action) {
			t.Errorf("%s is served anonymously but is not in AnonymousActions", route.Action)
		}
	}
	for _, route := range split.Gated {
		if slices.Contains(AnonymousActions, route.Action) {
			t.Errorf("%s is anonymous by declaration but sits in the gated half", route.Action)
		}
	}
	if got := len(PublicRoutes(&closing{})); got != len(AnonymousActions) {
		t.Errorf("PublicRoutes has %d routes, want %d, one per anonymous action", got, len(AnonymousActions))
	}
	if got := len(OperatorRoutes(&closing{})); got != len(split.Gated) {
		t.Errorf("OperatorRoutes has %d routes, want %d", got, len(split.Gated))
	}
}

func TestEveryAnonymousActionIsADeclaredOperation(t *testing.T) {
	built, err := Build(&closing{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	declared := map[string]bool{}
	for _, route := range built {
		declared[route.Action] = true
	}
	for _, action := range AnonymousActions {
		if !declared[action] {
			t.Errorf("anonymous action %s matches no route", action)
		}
	}
}

func TestNoTwoFormsRoutesShareAMethodAndPath(t *testing.T) {
	built, err := Build(&closing{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	seen := map[string]bool{}
	for _, route := range built {
		key := route.Method + " " + route.Path
		if seen[key] {
			t.Errorf("%s is declared twice", key)
		}
		seen[key] = true
	}
}

func TestAReachedOnlyMountStillAnswersTheSameStatusesAndRecordsTheCaller(t *testing.T) {
	mount := sessionMount()
	mount.ReachedOnly = true
	manager := &closing{}
	if rec := closeStatus(t, manager, mount); rec.Code != http.StatusOK || manager.closedBy != "the-session" {
		t.Errorf("reached-only close answered %d by %q, want 200 by the-session", rec.Code, manager.closedBy)
	}
	refused := closeStatus(t, &closing{err: forms.ErrFormNotFound}, mount)
	if refused.Code != http.StatusNotFound {
		t.Errorf("reached-only refusal answered %d, want 404", refused.Code)
	}
}

func TestAReachedOnlyMountWithNoCallerStillServesTheRoute(t *testing.T) {
	manager := &closing{}
	rec := closeStatus(t, manager, Mount{ReachedOnly: true})
	if rec.Code == http.StatusInternalServerError {
		t.Errorf("a missing caller crashed the route: %d %s", rec.Code, rec.Body.String())
	}
}

func TestGatedToolsLeaveOutEveryAnonymousActionAndKeepTheRest(t *testing.T) {
	tools, err := BuildGatedTools(&closing{}, sessionMount())
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("no tools")
	}
	names := map[string]bool{}
	for _, tool := range tools {
		if slices.Contains(AnonymousActions, tool.Action) {
			t.Errorf("tool %s exposes anonymous action %s", tool.Name, tool.Action)
		}
		if names[tool.Name] {
			t.Errorf("tool %s listed twice", tool.Name)
		}
		names[tool.Name] = true
	}
	if !names["forms_form_close"] {
		t.Errorf("forms_form_close is missing from the gated tools")
	}
}

func toolNamed(t *testing.T, tools []Tool, name string) Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("no tool %s", name)
	return Tool{}
}

func TestAReachedOnlyToolStillRunsTheManagerForTheCaller(t *testing.T) {
	mount := sessionMount()
	mount.ReachedOnly = true
	manager := &closing{}
	tools, err := BuildGatedTools(manager, mount)
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	tool := toolNamed(t, tools, "forms_form_close")
	var runErr error
	switch {
	case tool.Handle != nil:
		_, runErr = tool.Handle(context.Background(), dispatchRequest(`{"form_id":"f1"}`))
	case tool.Invoke != nil:
		_, runErr = tool.Invoke(context.Background(), json.RawMessage(`{"form_id":"f1"}`))
	default:
		t.Fatal("tool has neither Invoke nor Handle")
	}
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	if manager.calls != 1 || manager.closedBy != "the-session" {
		t.Errorf("manager reached %d times, closed by %q, want once by the-session", manager.calls, manager.closedBy)
	}
}

func TestAToolRefusalCarriesTheSentinelBack(t *testing.T) {
	manager := &closing{err: forms.ErrFormNotFound}
	tools, err := BuildGatedTools(manager, sessionMount())
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	tool := toolNamed(t, tools, "forms_form_close")
	var runErr error
	if tool.Handle != nil {
		_, runErr = tool.Handle(context.Background(), dispatchRequest(`{"form_id":"f1"}`))
	} else {
		_, runErr = tool.Invoke(context.Background(), json.RawMessage(`{"form_id":"f1"}`))
	}
	if runErr == nil {
		t.Fatal("a refused close returned no error")
	}
}

func dispatchRequest(arguments string) dispatch.Request {
	return dispatch.Request{Arguments: json.RawMessage(arguments)}
}
