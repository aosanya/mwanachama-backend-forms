package mwanachamaforms

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) MarkPaid(ctx context.Context, formID, memberID string, paid bool, markedBy string) (models.Submission, error) {
	held, err := m.GetSubmission(ctx, formID, memberID)
	if err != nil {
		return models.Submission{}, fmt.Errorf("MarkPaid: %w", err)
	}
	held.PaidAt, held.PaidBy = "", ""
	if paid {
		held.PaidAt, held.PaidBy = models.NowRFC3339(), markedBy
	}
	if err := m.replace(ctx, roleSubmission, held.ID, held); err != nil {
		return models.Submission{}, fmt.Errorf("MarkPaid: %w", err)
	}
	held.Paid = paid
	return held, nil
}
