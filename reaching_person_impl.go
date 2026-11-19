package mwanachamaforms

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ListReachingPerson(ctx context.Context, personID string) ([]models.Form, error) {
	if personID == "" {
		return []models.Form{}, nil
	}
	forms, err := listOf[models.Form](m, m.q(ctx, roleForm).Order("title"), roleForm)
	if err != nil {
		return nil, fmt.Errorf("ListReachingPerson: %w", err)
	}
	out := make([]models.Form, 0, len(forms))
	for _, f := range forms {
		if !models.VisibleToMembers(f.Status) {
			continue
		}
		reaches, err := m.reaches(ctx, personID, f)
		if err != nil {
			return nil, fmt.Errorf("ListReachingPerson: %w", err)
		}
		if reaches {
			out = append(out, f)
		}
	}
	return out, nil
}
