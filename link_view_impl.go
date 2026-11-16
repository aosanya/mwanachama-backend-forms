package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ShowPublicLink(ctx context.Context, key string) (models.ResolvedLinkView, error) {
	opened, err := m.OpenPublicLink(ctx, key)
	if err != nil {
		return models.ResolvedLinkView{}, err
	}
	return opened.RespondentView(), nil
}
