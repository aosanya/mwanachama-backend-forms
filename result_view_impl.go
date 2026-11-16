package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ShowLatestResults(ctx context.Context, formID, callerID string) (models.ResultReportView, error) {
	report, err := m.LatestResults(ctx, formID, callerID)
	if err != nil {
		return models.ResultReportView{}, err
	}
	return report.RespondentView(), nil
}
