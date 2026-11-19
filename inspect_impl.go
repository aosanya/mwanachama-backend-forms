package mwanachamaforms

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"strings"
)

func validInspectionResult(result models.InspectionResult) bool {
	return result == models.InspectionComplies || result == models.InspectionDoesNotComply || result == models.InspectionNeedsMore
}
func (m *formManager) InspectSubmission(ctx context.Context, formID, memberID string, result models.InspectionResult, note, inspectedBy string) (models.Submission, error) {
	note = strings.TrimSpace(note)
	if inspectedBy == "" || !validInspectionResult(result) {
		return models.Submission{}, ErrInvalidInspection
	}
	if result != models.InspectionComplies && note == "" {
		return models.Submission{}, ErrMissingNote
	}
	if inspectedBy == memberID {
		return models.Submission{}, ErrSelfInspection
	}
	held, err := m.GetSubmission(ctx, formID, memberID)
	if err != nil {
		return models.Submission{}, fmt.Errorf("InspectSubmission: %w", err)
	}
	held.InspectionResult, held.InspectionNote, held.InspectedBy, held.InspectedAt = result, note, inspectedBy, models.NowRFC3339()
	if err := m.replace(ctx, roleSubmission, held.ID, held); err != nil {
		return models.Submission{}, fmt.Errorf("InspectSubmission: %w", err)
	}
	return held, nil
}
func withoutFinding(held models.Submission) models.Submission {
	held.InspectionResult, held.InspectionNote, held.InspectedBy, held.InspectedAt = "", "", "", ""
	return held
}
