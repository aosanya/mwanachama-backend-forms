package mwanachamaforms_test

import (
	"context"
	"encoding/json"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"strings"
	"testing"
)

func sendOn(t *testing.T, m forms.FormManager) voiceRound {
	t.Helper()
	r := openRoundForTwoGroups(t, m, models.OneVoiceOff)
	if _, err := m.SubmitAnswersAs(context.Background(), r.form.ID, "mem-1", "g-1", []models.Answer{{QuestionID: r.question.ID, ValueText: "fence on Hársfa utca"}, {QuestionID: r.reason.ID, ValueText: "new"}}); err != nil {
		t.Fatalf("send: %v", err)
	}
	return r
}
func inspectsOnly(callerID string, formIDs ...string) forms.InspectorForms {
	return func(_ context.Context, asked string) ([]string, error) {
		if asked != callerID {
			return nil, nil
		}
		return formIDs, nil
	}
}
func TestThePlanningOfficersListShowsTheFencePermitSendsAndNotTheTerraceLicences(t *testing.T) {
	ctx := context.Background()
	m := gatedManager(t, nil)
	fence := sendOn(t, m)
	sendOn(t, m)
	m.UseInspectorForms(inspectsOnly("planning-officer", fence.form.ID))
	got, err := m.ListInspectableSubmissions(ctx, "planning-officer")
	if err != nil {
		t.Fatalf("ListInspectableSubmissions: %v", err)
	}
	if len(got) != 1 || got[0].FormID != fence.form.ID {
		t.Fatalf("list = %+v, want only the fence permit's send", got)
	}
}
func TestACallerWhoInspectsNoPermitSeesNoSends(t *testing.T) {
	ctx := context.Background()
	m := gatedManager(t, nil)
	fence := sendOn(t, m)
	m.UseInspectorForms(inspectsOnly("planning-officer", fence.form.ID))
	got, err := m.ListInspectableSubmissions(ctx, "clerk-1")
	if err != nil || len(got) != 0 {
		t.Fatalf("list = %+v (%v), want none", got, err)
	}
}
func TestWithNoInspectorSourceTheInspectorListIsEmpty(t *testing.T) {
	m := gatedManager(t, nil)
	sendOn(t, m)
	got, err := m.ListInspectableSubmissions(context.Background(), "planning-officer")
	if err != nil || len(got) != 0 {
		t.Fatalf("list = %+v (%v), want none", got, err)
	}
}
func TestTheFormResultsListStillReadsEveryFormsSends(t *testing.T) {
	ctx := context.Background()
	m := gatedManager(t, nil)
	fence := sendOn(t, m)
	m.UseInspectorForms(inspectsOnly("planning-officer", fence.form.ID))
	got, err := m.ListSubmissions(ctx, fence.form.ID, "", "", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("ListSubmissions = %+v (%v), want the send", got, err)
	}
}
func TestTheInspectorListIsReadOverTheRouteForTheCallerNotTheBody(t *testing.T) {
	m := gatedManager(t, nil)
	fence := sendOn(t, m)
	sendOn(t, m)
	m.UseInspectorForms(inspectsOnly("planning-officer", fence.form.ID))
	status, raw := callRoute(t, fence, "planning-officer", "forms.submission.list_inspectable", "", nil)
	var rows []map[string]any
	if status != 200 || json.Unmarshal(raw, &rows) != nil || len(rows) != 1 || rows[0]["form_id"] != fence.form.ID {
		t.Fatalf("inspector list = %d %s, want the fence permit's one send", status, raw)
	}
	status, raw = callRoute(t, fence, "clerk-1", "forms.submission.list_inspectable", "", nil)
	if status != 200 || strings.TrimSpace(string(raw)) != "[]" && strings.TrimSpace(string(raw)) != "null" {
		t.Fatalf("a caller with no inspector role = %d %s, want none", status, raw)
	}
}
