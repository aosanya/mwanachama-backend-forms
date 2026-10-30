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
	opened, err := m.OpenPublicLink(unnarrowed(ctx), key)
	if err != nil {
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
		return m.claimedThrough(ctx, held, opened.Form.ID)
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
		return m.claimedMeanwhile(ctx, publicKey, memberID, opened.Form.ID)
	}
	return m.claimedThrough(ctx, held, opened.Form.ID)
}

func (m *formManager) claimedThrough(ctx context.Context, held models.Respondent, formID string) (models.RespondentPublicView, error) {
	recorded, err := m.claimRecorded(ctx, held.ID, formID)
	if err != nil {
		return models.RespondentPublicView{}, fmt.Errorf("ClaimRespondent: %w", err)
	}
	if recorded {
		return held.PublicView(), nil
	}
	err = m.insert(ctx, roleClaim, models.Claim{
		ID: newID(), RespondentID: held.ID, FormID: formID, ClaimedAt: models.NowRFC3339(),
	})
	if err != nil {
		if recorded, _ := m.claimRecorded(ctx, held.ID, formID); recorded {
			return held.PublicView(), nil
		}
		return models.RespondentPublicView{}, fmt.Errorf("ClaimRespondent: %w", err)
	}
	return held.PublicView(), nil
}

func (m *formManager) claimRecorded(ctx context.Context, respondentID, formID string) (bool, error) {
	var count int64
	err := m.q(ctx, roleClaim).Where("respondent_id = ? AND form_id = ?", respondentID, formID).Count(&count).Error
	return count > 0, err
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

func (m *formManager) claimedMeanwhile(ctx context.Context, publicKey, memberID, formID string) (models.RespondentPublicView, error) {
	held, err := m.respondentHolding(ctx, publicKey)
	if err != nil {
		return models.RespondentPublicView{}, err
	}
	if held.ClaimedByMemberID != memberID {
		return models.RespondentPublicView{}, ErrRespondentClaimed
	}
	return m.claimedThrough(ctx, held, formID)
}

func (m *formManager) cameByLink(ctx context.Context, callerID, formID string) (bool, error) {
	claimed := m.from(m.db, roleRespondent).Select("id").Where("claimed_by_member_id = ?", callerID)
	for _, role := range []string{roleClaim, roleDeclaration} {
		var held int64
		err := m.q(ctx, role).
			Where("form_id = ? AND respondent_id IN (?)", formID, claimed).
			Count(&held).Error
		if err != nil || held > 0 {
			return held > 0, err
		}
	}
	return false, nil
}
