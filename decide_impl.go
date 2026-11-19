package mwanachamaforms

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"strings"
	"time"
)

const dayLayout = "2006-01-02"

func grantFields(grant []string) (certificateRef, validUntil string) {
	if len(grant) > 0 {
		certificateRef = strings.TrimSpace(grant[0])
	}
	if len(grant) > 1 {
		validUntil = strings.TrimSpace(grant[1])
	}
	return certificateRef, validUntil
}
func checkGrant(decision models.Decision, certificateRef, validUntil string) error {
	if certificateRef == "" && validUntil == "" {
		return nil
	}
	if decision != models.DecisionApproved {
		return ErrGrantOnRefusal
	}
	if validUntil == "" {
		return nil
	}
	day, err := time.Parse(dayLayout, validUntil)
	if err != nil || day.Format(dayLayout) != validUntil {
		return ErrInvalidValidUntil
	}
	if validUntil < models.NowRFC3339()[:len(dayLayout)] {
		return ErrValidUntilPast
	}
	return nil
}
func (m *formManager) DecideSubmission(ctx context.Context, formID, memberID string, decision models.Decision, note, decidedBy string, grant ...string) (models.Submission, error) {
	note = strings.TrimSpace(note)
	certificateRef, validUntil := grantFields(grant)
	if decidedBy == "" || (decision != models.DecisionApproved && decision != models.DecisionRefused) {
		return models.Submission{}, ErrInvalidDecision
	}
	if decision == models.DecisionRefused && note == "" {
		return models.Submission{}, ErrMissingNote
	}
	if decidedBy == memberID {
		return models.Submission{}, ErrSelfDecision
	}
	if err := checkGrant(decision, certificateRef, validUntil); err != nil {
		return models.Submission{}, err
	}
	held, err := m.GetSubmission(ctx, formID, memberID)
	if err != nil {
		return models.Submission{}, fmt.Errorf("DecideSubmission: %w", err)
	}
	held.Decision, held.DecidedBy, held.DecidedAt, held.DecisionNote = decision, decidedBy, models.NowRFC3339(), note
	held.CertificateRef, held.ValidUntil = certificateRef, validUntil
	if err := m.replace(ctx, roleSubmission, held.ID, held); err != nil {
		return models.Submission{}, fmt.Errorf("DecideSubmission: %w", err)
	}
	return held, nil
}
