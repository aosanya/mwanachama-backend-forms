package routes

import (
	"context"
	"encoding/json"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"strings"
	"testing"
)

type inspectors struct {
	forms.FormManager
	formID, memberID, note, inspectedBy string
	result                              models.InspectionResult
}

func (i *inspectors) InspectSubmission(_ context.Context, formID, memberID string, result models.InspectionResult, note, inspectedBy string) (models.Submission, error) {
	i.formID, i.memberID, i.result, i.note, i.inspectedBy = formID, memberID, result, note, inspectedBy
	return models.Submission{FormID: formID, MemberID: memberID, InspectionResult: result, InspectionNote: note, InspectedBy: inspectedBy}, nil
}
func TestAFindingIsRecordedByTheCallerNotTheBodyThroughTheTool(t *testing.T) {
	manager := &inspectors{}
	tools, err := BuildGatedTools(manager, sessionMount())
	if err != nil {
		t.Fatalf("BuildGatedTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name != "forms_submission_inspect" {
			continue
		}
		if tool.Action != "forms.submission.inspect" {
			t.Fatalf("action = %q", tool.Action)
		}
		var schema struct {
			Required   []string       `json:"required"`
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("schema: %v", err)
		}
		if _, asked := schema.Properties["inspected_by"]; asked {
			t.Errorf("the tool asks for inspected_by: %s", tool.InputSchema)
		}
		for _, want := range []string{"form_id", "member_id", "result"} {
			if !strings.Contains(strings.Join(schema.Required, ","), want) {
				t.Errorf("%s is not required: %s", want, tool.InputSchema)
			}
		}
		if _, err := tool.Invoke(context.Background(), json.RawMessage(`{"form_id":"f1","member_id":"mem-1","result":"needs_more","note":"site plan missing"}`)); err != nil {
			t.Fatalf("Invoke: %v", err)
		}
		if manager.inspectedBy != "the-session" || manager.formID != "f1" || manager.memberID != "mem-1" || manager.result != models.InspectionNeedsMore || manager.note != "site plan missing" {
			t.Errorf("inspected = %+v, want the caller the-session and the named send", manager)
		}
		if _, err := tool.Invoke(context.Background(), json.RawMessage(`{"form_id":"f1","member_id":"mem-1","result":"complies","inspected_by":"forged-inspector"}`)); err == nil {
			t.Error("a forged inspected_by was accepted")
		}
		return
	}
	t.Fatal("no forms_submission_inspect tool")
}
