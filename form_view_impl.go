package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ShowIn(ctx context.Context, formID, language string) (models.FormRead, error) {
	f, err := m.GetIn(ctx, formID, language)
	if err != nil {
		return nil, err
	}
	if _, narrowed := narrowedTo(ctx); narrowed {
		return f.RespondentView(), nil
	}
	return f, nil
}
