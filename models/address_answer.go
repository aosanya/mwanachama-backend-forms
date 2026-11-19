package models

import (
	"fmt"
	"github.com/aosanya/mwanachama-backend-shared/answerrules"
	"strings"
)

func checkAddress(q Question, a Answer) error {
	if strings.TrimSpace(a.ValueText) == "" {
		return fmt.Errorf("address answer requires value_text")
	}
	if a.PlaceRef == "" {
		if !q.AllowFreeText {
			return fmt.Errorf("address answer requires place_ref, because this question does not allow free text")
		}
		return nil
	}
	if !placeRefFits(a.PlaceRef) {
		return fmt.Errorf("place_ref %q must be address: or street: followed by what was picked", a.PlaceRef)
	}
	return nil
}
func placeRefFits(ref string) bool {
	return answerrules.PlaceRefFits(ref)
}
