package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) ClaimRespondent(ctx context.Context, key, publicKey, memberID string) (models.RespondentPublicView, error) {
	if memberID == "" {
		return models.RespondentPublicView{}, fmt.Errorf("%w: nobody is signed in to claim it", ErrInvalidReference)
	}
	if _, err := m.OpenPublicLink(unnarrowed(ctx), key); err != nil {
		return models.RespondentPublicView{}, err
	}
	if publicKey == "" {
		return models.RespondentPublicView{}, fmt.Errorf("%w: public_key is required", ErrInvalidReference)
	}
	held, err := m.respondentHolding(ctx, publicKey)
	if err != nil {
		return models.RespondentPublicView{}, err
	}
	switch held.ClaimedByMemberID {
	case memberID:
		return held.PublicView(), nil
	case "":
	default:
		return models.RespondentPublicView{}, ErrRespondentClaimed
	}

	held.ClaimedByMemberID, held.ClaimedAt = memberID, models.NowRFC3339()
	row, err := encode(m.st.Object(roleRespondent), held)
	if err != nil {
		return models.RespondentPublicView{}, fmt.Errorf("ClaimRespondent: %w", err)
	}
	res := m.q(ctx, roleRespondent).
		Where("id = ? AND (claimed_by_member_id = '' OR claimed_by_member_id IS NULL)", held.ID).
		Updates(row)
	if res.Error != nil {
		return models.RespondentPublicView{}, fmt.Errorf("ClaimRespondent: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return m.claimedMeanwhile(ctx, publicKey, memberID)
	}
	return held.PublicView(), nil
}

func (m *formManager) respondentHolding(ctx context.Context, publicKey string) (models.Respondent, error) {
	var held models.Respondent
	err := m.take(m.q(ctx, roleRespondent).Where("public_key = ?", publicKey), roleRespondent, &held, errNoRespondent)
	if errors.Is(err, errNoRespondent) {
		return models.Respondent{}, fmt.Errorf("%w: no respondent holds that public_key", ErrInvalidReference)
	}
	if err != nil {
		return models.Respondent{}, fmt.Errorf("ClaimRespondent: %w", err)
	}
	return held, nil
}

func (m *formManager) claimedMeanwhile(ctx context.Context, publicKey, memberID string) (models.RespondentPublicView, error) {
	held, err := m.respondentHolding(ctx, publicKey)
	if err != nil {
		return models.RespondentPublicView{}, err
	}
	if held.ClaimedByMemberID != memberID {
		return models.RespondentPublicView{}, ErrRespondentClaimed
	}
	return held.PublicView(), nil
}

func (m *formManager) cameByLink(ctx context.Context, callerID, formID string) (bool, error) {
	claimed := m.db.Table(m.table(roleRespondent)).Select("id").Where("claimed_by_member_id = ?", callerID)
	var declared int64
	err := m.q(ctx, roleDeclaration).
		Where("form_id = ? AND respondent_id IN (?)", formID, claimed).
		Count(&declared).Error
	return declared > 0, err
}
