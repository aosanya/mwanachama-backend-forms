package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ShowMySubmission(ctx context.Context, formID, memberID string) (models.Submission, error) {
	held, err := m.GetSubmission(ctx, formID, memberID)
	held.PaidBy = ""
	held = withoutFinding(held)
	return held, err
}
