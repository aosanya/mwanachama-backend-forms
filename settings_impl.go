package mwanachamaforms

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	SettingApprovalRequired = "approval_required"
	SettingFormTypes        = "form_types"
)

type resolvedSettings struct {
	approvalRequired bool
	formTypes        []models.FormType
}

func resolveSettings(db *gorm.DB, s *spec.Spec) (resolvedSettings, error) {
	approval, err := spec.ResolveStored(db, s, SettingApprovalRequired)
	if err != nil {
		return resolvedSettings{}, err
	}
	required, ok := approval.Value.(bool)
	if !ok {
		return resolvedSettings{}, fmt.Errorf("%s: %v is not a bool", SettingApprovalRequired, approval.Value)
	}
	typesSetting, err := spec.ResolveStored(db, s, SettingFormTypes)
	if err != nil {
		return resolvedSettings{}, err
	}
	types, err := models.ParseFormTypes(typesSetting.Value)
	if err != nil {
		return resolvedSettings{}, err
	}
	return resolvedSettings{approvalRequired: required, formTypes: types}, nil
}

func (m *formManager) FormTypes() []models.FormType {
	return append([]models.FormType(nil), m.formTypes...)
}
