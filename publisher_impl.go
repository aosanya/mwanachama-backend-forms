package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type Names func(ctx context.Context, callerID string) (string, error)

func (m *formManager) UseNames(n Names) { m.names = n }

func (m *formManager) nameOf(ctx context.Context, callerID string) (string, error) {
	if m.names == nil || callerID == "" {
		return "", nil
	}
	name, err := m.names(ctx, callerID)
	if err != nil {
		return "", err
	}
	return models.ShortenPublisherName(name), nil
}
