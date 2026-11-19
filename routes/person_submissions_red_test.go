package routes

import (
	"context"
	"encoding/json"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

const (
	listOfPersonAction = "forms.submission.list_of_person"
	listOfPersonTool   = "forms_submission_list_of_person"
)

type personLister struct {
	forms.FormManager
	asked []string
}

func (p *personLister) ListSubmissionsOfPerson(_ context.Context, memberID, _ string) ([]models.Submission, error) {
	p.asked = append(p.asked, memberID)
	return []models.Submission{{ID: "s-2", FormID: "f-2", MemberID: memberID, Answers: 1}, {ID: "s-1", FormID: "f-1", MemberID: memberID, Answers: 2}}, nil
}
func personToolOf(t *testing.T, manager forms.FormManager) Tool {
	t.Helper()
	tools, err := BuildGatedTools(manager, sessionMount())
	if err != nil {
		t.Fatalf("BuildGatedTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Action == listOfPersonAction {
			return tool
		}
	}
	t.Fatalf("the forms tools have no tool for %s", listOfPersonAction)
	return Tool{}
}
func TestThePersonSendsToolRequiresAMemberAndTakesNoForm(t *testing.T) {
	tool := personToolOf(t, &personLister{})
	if tool.Name != listOfPersonTool {
		t.Errorf("tool name = %q, want %q", tool.Name, listOfPersonTool)
	}
	var schema struct {
		Required   []string       `json:"required"`
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if !slices.Contains(schema.Required, "member_id") {
		t.Errorf("member_id is not required: %s", tool.InputSchema)
	}
	if _, asked := schema.Properties["form_id"]; asked {
		t.Errorf("the tool asks for a form_id: %s", tool.InputSchema)
	}
}
func TestThePersonSendsToolListsTheMemberNamedAcrossForms(t *testing.T) {
	manager := &personLister{}
	tool := personToolOf(t, manager)
	out, err := tool.Invoke(context.Background(), json.RawMessage(`{"member_id":"mem-7"}`))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if !slices.Equal(manager.asked, []string{"mem-7"}) {
		t.Errorf("ListSubmissionsOfPerson was asked for %v, want mem-7 alone", manager.asked)
	}
	raw, _ := json.Marshal(out)
	var rows []models.Submission
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) != 2 || rows[0].ID != "s-2" {
		t.Errorf("tool answered %s (%v), want the manager's two sends in its order", raw, err)
	}
	if _, err := tool.Invoke(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("a missing member_id was answered")
	}
}
func TestGetSubmissionsAnswersThePersonsSendsAcrossForms(t *testing.T) {
	manager := &personLister{}
	var route Route
	for _, candidate := range RoutesFor(manager, sessionMount()) {
		if candidate.Action == listOfPersonAction {
			route = candidate
		}
	}
	if route.Handler == nil {
		t.Fatalf("the forms route table has no route for %s", listOfPersonAction)
	}
	if route.Method != http.MethodGet || route.Path != "/submissions" {
		t.Fatalf("route = %s %s, want GET /submissions", route.Method, route.Path)
	}
	rec := httptest.NewRecorder()
	route.Handler(rec, httptest.NewRequest(http.MethodGet, "/submissions?member_id=mem-7", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /submissions?member_id=mem-7 answered %d %s", rec.Code, rec.Body.String())
	}
	var rows []models.Submission
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil || len(rows) != 2 || rows[0].MemberID != "mem-7" {
		t.Errorf("body = %s (%v), want the two sends of mem-7", rec.Body.String(), err)
	}
	if !slices.Equal(manager.asked, []string{"mem-7"}) {
		t.Errorf("ListSubmissionsOfPerson was asked for %v, want mem-7 alone", manager.asked)
	}
}
func TestGetSubmissionsWithoutAMemberIsRefusedBeforeTheManager(t *testing.T) {
	manager := &personLister{}
	var route Route
	for _, candidate := range RoutesFor(manager, sessionMount()) {
		if candidate.Action == listOfPersonAction {
			route = candidate
		}
	}
	if route.Handler == nil {
		t.Fatalf("the forms route table has no route for %s", listOfPersonAction)
	}
	for _, target := range []string{"/submissions", "/submissions?member_id="} {
		rec := httptest.NewRecorder()
		route.Handler(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s answered %d %s, want 400", target, rec.Code, rec.Body.String())
		}
	}
	if len(manager.asked) != 0 {
		t.Errorf("the manager was asked for %v on a request without a member", manager.asked)
	}
}
