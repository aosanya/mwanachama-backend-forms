package mwanachamaforms

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type InspectorForms func(ctx context.Context, inspectorID string) ([]string, error)

func (m *formManager) UseInspectorForms(f InspectorForms) {
	m.inspectorForms = f
}
func (m *formManager) ListInspectableSubmissions(ctx context.Context, inspectorID string) ([]models.Submission, error) {
	if m.inspectorForms == nil || inspectorID == "" {
		return nil, nil
	}
	formIDs, err := m.inspectorForms(ctx, inspectorID)
	if err != nil {
		return nil, fmt.Errorf("ListInspectableSubmissions: %w", err)
	}
	var out []models.Submission
	for _, formID := range formIDs {
		held, err := m.ListSubmissions(ctx, formID, "", "", "")
		if err != nil {
			return nil, fmt.Errorf("ListInspectableSubmissions: %w", err)
		}
		out = append(out, held...)
	}
	return out, nil
}
