package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type authors struct {
	forms.FormManager
	submittedBy, withdrawnBy, closedBy string
}

func (a *authors) Submit(_ context.Context, _, by string) (models.Form, error) {
	a.submittedBy = by
	return models.Form{}, nil
}

func (a *authors) Withdraw(_ context.Context, _, by string) (models.Form, error) {
	a.withdrawnBy = by
	return models.Form{}, nil
}

func (a *authors) Close(_ context.Context, _, by string) (models.Form, error) {
	a.closedBy = by
	return models.Form{}, nil
}

func sessionMount() Mount {
	return Mount{Caller: func(context.Context) string { return "the-session" }}
}

func postTo(t *testing.T, manager forms.FormManager, action, body string) {
	t.Helper()
	for _, route := range RoutesFor(manager, sessionMount()) {
		if route.Action != action {
			continue
		}
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("form_id", "f1")
		route.Handler(httptest.NewRecorder(), req)
		return
	}
	t.Fatalf("no route for %s", action)
}

func TestAFormIsSentForADecisionByTheCallerNotTheBody(t *testing.T) {
	manager := &authors{}
	postTo(t, manager, "forms.form.submit", `{"submitted_by":"forged-clerk"}`)
	if manager.submittedBy != "the-session" {
		t.Errorf("the form names %q as its sender; want the caller's own id the-session, never the body's forged-clerk", manager.submittedBy)
	}
}

func TestAFormIsWithdrawnByTheCallerNotTheBody(t *testing.T) {
	manager := &authors{}
	postTo(t, manager, "forms.form.withdraw", `{"actor_id":"forged-clerk"}`)
	if manager.withdrawnBy != "the-session" {
		t.Errorf("the refused round names %q as who pulled it back; want the caller's own id the-session, never the body's forged-clerk", manager.withdrawnBy)
	}
}

func TestAFormIsClosedByTheCallerNotTheBody(t *testing.T) {
	manager := &authors{}
	postTo(t, manager, "forms.form.close", `{"closed_by":"forged-clerk"}`)
	if manager.closedBy != "the-session" {
		t.Errorf("the form names %q as who closed it; want the caller's own id the-session, never the body's forged-clerk", manager.closedBy)
	}
}

func TestFormToolsDoNotAskForTheirAuthor(t *testing.T) {
	tools, err := BuildGatedTools(&authors{}, sessionMount())
	if err != nil {
		t.Fatalf("BuildGatedTools: %v", err)
	}
	for tool, field := range map[string]string{"forms_form_submit": "submitted_by", "forms_form_withdraw": "actor_id", "forms_form_close": "closed_by"} {
		found := false
		for _, candidate := range tools {
			if candidate.Name != tool {
				continue
			}
			found = true
			var schema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			}
			if err := json.Unmarshal(candidate.InputSchema, &schema); err != nil {
				t.Fatalf("%s schema: %v", tool, err)
			}
			if _, asked := schema.Properties[field]; asked {
				t.Errorf("%s still takes %s from the request; the caller signs it", tool, field)
			}
		}
		if !found {
			t.Fatalf("%s is not a tool", tool)
		}
	}
}
