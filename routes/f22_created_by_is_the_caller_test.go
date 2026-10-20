package routes_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestComposingAFormRecordsTheCallerAsItsCreatorWhateverTheBodyClaims(t *testing.T) {
	fm := newTestManager(t)
	h := mountAsCaller(t, fm)
	r := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(`{
		"title": "Census",
		"closes_at": "2099-01-01T00:00:00Z",
		"created_by": "impostor"
	}`))
	r.Header.Set("X-Test-Caller", "clerk-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var f mwanachamaforms.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.CreatedBy != "clerk-1" {
		t.Fatalf("create answered created_by %q, want the caller clerk-1", f.CreatedBy)
	}
	if got := getFormAs(t, h, "clerk-1", f.ID); got.CreatedBy != "clerk-1" {
		t.Fatalf("the stored form says created_by %q, want the caller clerk-1", got.CreatedBy)
	}
}

func TestTheComposeToolTakesNoCreator(t *testing.T) {
	s, err := dispatch.Parse(mwanachamaforms.Operations())
	if err != nil {
		t.Fatalf("parse operations: %v", err)
	}
	named := map[string]error{}
	for name := range s.Errors {
		named[name] = errors.New(name)
	}
	tools, err := dispatch.Tools(s, dispatch.Deps{Manager: newTestManager(t), Errors: named})
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	for _, tool := range tools {
		if tool.Action != "forms.form.create" {
			continue
		}
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("schema: %v", err)
		}
		if _, ok := schema.Properties["created_by"]; ok {
			t.Fatalf("%s offers created_by: %s", tool.Name, tool.InputSchema)
		}
		if _, ok := schema.Properties["title"]; !ok {
			t.Fatalf("%s lost the body it composes from: %s", tool.Name, tool.InputSchema)
		}
		return
	}
	t.Fatal("no tool carries forms.form.create")
}
