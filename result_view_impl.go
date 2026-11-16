package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ShowLatestResults(ctx context.Context, formID, callerID string) (models.ResultReportRead, error) {
	report, err := m.LatestResults(ctx, formID, callerID)
	if err != nil {
		return nil, err
	}
	return report.ReadAs(IsNarrowed(ctx)), nil
}
