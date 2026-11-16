package mwanachamaforms

import (
	"context"
	"fmt"
)

func (m *formManager) ReleaseMember(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w: nobody is withdrawing", ErrInvalidAnswer)
	}
	err := m.q(ctx, roleRespondent).Where("claimed_by_member_id = ?", memberID).
		Updates(map[string]any{"claimed_by_member_id": "", "claimed_at": nil}).Error
	if err != nil {
		return fmt.Errorf("ReleaseMember: %w", err)
	}
	return nil
}
