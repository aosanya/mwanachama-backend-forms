package routes

import (
	"context"
	"encoding/json"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"strings"
	"testing"
)

type deciders struct {
	forms.FormManager
	formID, memberID, note, decidedBy string
	decision                          models.Decision
}

func (d *deciders) DecideSubmission(_ context.Context, formID, memberID string, decision models.Decision, note, decidedBy string, _ ...string) (models.Submission, error) {
	d.formID, d.memberID, d.decision, d.note, d.decidedBy = formID, memberID, decision, note, decidedBy
	return models.Submission{FormID: formID, MemberID: memberID, Decision: decision, DecisionNote: note, DecidedBy: decidedBy}, nil
}
func TestASendIsDecidedByTheCallerNotTheBodyThroughTheTool(t *testing.T) {
	manager := &deciders{}
	tools, err := BuildGatedTools(manager, sessionMount())
	if err != nil {
		t.Fatalf("BuildGatedTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name != "forms_submission_decide" {
			continue
		}
		if tool.Action != "forms.submission.decide" {
			t.Fatalf("action = %q", tool.Action)
		}
		var schema struct {
			Required   []string       `json:"required"`
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("schema: %v", err)
		}
		if _, asked := schema.Properties["decided_by"]; asked {
			t.Errorf("the tool asks for decided_by: %s", tool.InputSchema)
		}
		for _, want := range []string{"form_id", "member_id", "decision"} {
			if !strings.Contains(strings.Join(schema.Required, ","), want) {
				t.Errorf("%s is not required: %s", want, tool.InputSchema)
			}
		}
		if _, err := tool.Invoke(context.Background(), json.RawMessage(`{"form_id":"f1","member_id":"mem-1","decision":"refused","note":"plan missing"}`)); err != nil {
			t.Fatalf("Invoke: %v", err)
		}
		if manager.decidedBy != "the-session" || manager.formID != "f1" || manager.memberID != "mem-1" || manager.decision != models.DecisionRefused || manager.note != "plan missing" {
			t.Errorf("decided = %+v, want the caller the-session and the named send", manager)
		}
		if _, err := tool.Invoke(context.Background(), json.RawMessage(`{"form_id":"f1","member_id":"mem-1","decision":"approved","decided_by":"forged-clerk"}`)); err == nil {
			t.Error("a forged decided_by was accepted")
		}
		return
	}
	t.Fatal("no forms_submission_decide tool")
}
func TestTheDecideToolOffersTheCertificateAndTheEndDayAsOptionalInputs(t *testing.T) {
	tools, err := BuildGatedTools(&deciders{}, sessionMount())
	if err != nil {
		t.Fatalf("BuildGatedTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name != "forms_submission_decide" {
			continue
		}
		var schema struct {
			Required   []string       `json:"required"`
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("schema: %v", err)
		}
		for _, name := range []string{"certificate_ref", "valid_until"} {
			if _, offered := schema.Properties[name]; !offered {
				t.Errorf("the tool does not offer %s: %s", name, tool.InputSchema)
			}
			if strings.Contains(strings.Join(schema.Required, ","), name) {
				t.Errorf("%s is required: %s", name, tool.InputSchema)
			}
		}
		return
	}
	t.Fatal("no forms_submission_decide tool")
}
