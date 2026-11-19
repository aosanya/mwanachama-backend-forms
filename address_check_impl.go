package mwanachamaforms

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type AddressCheck func(ctx context.Context, placeRef string) (bool, error)

func (m *formManager) UseAddressCheck(c AddressCheck) {
	m.addressCheck = c
}
func (m *formManager) checkPlaceRefs(ctx context.Context, in []models.Answer) error {
	if m.addressCheck == nil {
		return nil
	}
	for _, a := range in {
		if a.PlaceRef == "" {
			continue
		}
		held, err := m.addressCheck(ctx, a.PlaceRef)
		if err != nil {
			return fmt.Errorf("checking place_ref: %w", err)
		}
		if !held {
			return fmt.Errorf("%w: question %s: place_ref %q is not a street or address the town holds", ErrInvalidAnswer, a.QuestionID, a.PlaceRef)
		}
	}
	return nil
}
