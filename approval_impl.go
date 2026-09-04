// approval_impl.go — the submit/withdraw/approve/refuse workflow. Ported
// from mwanachama-backend-api-gateway's
// internal/store/memory/survey_store_approval.go.
package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/gormstore"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// openApproval returns the form's undecided approval row, if one exists.
func (m *formManager) openApproval(ctx context.Context, formID string) (gormstore.ApprovalRow, bool, error) {
	var row gormstore.ApprovalRow
	err := m.db.WithContext(ctx).Table(m.tables.Approvals).Where("form_id = ? AND decided_at = ?", formID, "").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ApprovalRow{}, false, nil
	}
	if err != nil {
		return gormstore.ApprovalRow{}, false, err
	}
	return row, true, nil
}

// Submit moves a draft Form to submitted and opens a new undecided Approval.
func (m *formManager) Submit(ctx context.Context, id, submittedBy string) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusDraft {
		return models.Form{}, ErrNotDraft
	}
	now := models.NowRFC3339()
	err = m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).
		Updates(map[string]any{"status": string(models.StatusSubmitted), "submitted_at": now, "submitted_by": submittedBy, "updated_at": now}).Error
	if err != nil {
		return models.Form{}, fmt.Errorf("Submit: %w", err)
	}
	arow := gormstore.ApprovalToRow(models.Approval{FormID: id, RequestedBy: submittedBy, RequestedAt: now})
	if err := m.db.WithContext(ctx).Table(m.tables.Approvals).Create(&arow).Error; err != nil {
		return models.Form{}, fmt.Errorf("Submit: %w", err)
	}
	f.Status, f.SubmittedAt, f.SubmittedBy, f.LastUpdated = models.StatusSubmitted, now, submittedBy, now
	return f, nil
}

// Withdraw reverts a submitted or approved Form to draft, closing its open
// approval as refused.
func (m *formManager) Withdraw(ctx context.Context, id, actorID string) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusSubmitted && f.Status != models.StatusApproved {
		return models.Form{}, ErrNotWithdrawable
	}
	now := models.NowRFC3339()
	err = m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).
		Updates(map[string]any{
			"status":       string(models.StatusDraft),
			"submitted_at": "", "submitted_by": "",
			"approved_at": "", "approved_by": "",
			"updated_at": now,
		}).Error
	if err != nil {
		return models.Form{}, fmt.Errorf("Withdraw: %w", err)
	}

	row, found, err := m.openApproval(ctx, id)
	if err != nil {
		return models.Form{}, fmt.Errorf("Withdraw: %w", err)
	}
	if found {
		err = m.db.WithContext(ctx).Table(m.tables.Approvals).Where("id = ?", row.ID).
			Updates(map[string]any{
				"decided_by": actorID, "decided_at": now,
				"decision": string(models.DecisionRefused),
				"note":     "withdrawn by its author before a decision",
			}).Error
		if err != nil {
			return models.Form{}, fmt.Errorf("Withdraw: %w", err)
		}
	}

	f.Status = models.StatusDraft
	f.SubmittedAt, f.SubmittedBy = "", ""
	f.ApprovedAt, f.ApprovedBy = "", ""
	f.LastUpdated = now
	return f, nil
}

// decide is Approve/Refuse's shared body: requires a submitted form, moves
// it to approved or back to draft, and stamps the open approval — or, in the
// defensive case no open approval exists (should not happen from Submit),
// synthesizes one first.
func (m *formManager) decide(ctx context.Context, id, approverID, note string, decision models.Decision) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusSubmitted {
		return models.Form{}, ErrNotSubmitted
	}
	submittedBy := f.SubmittedBy

	now := models.NowRFC3339()
	updates := map[string]any{"updated_at": now}
	if decision == models.DecisionApproved {
		updates["status"], updates["approved_at"], updates["approved_by"] = string(models.StatusApproved), now, approverID
		f.Status, f.ApprovedAt, f.ApprovedBy = models.StatusApproved, now, approverID
	} else {
		updates["status"], updates["submitted_at"], updates["submitted_by"] = string(models.StatusDraft), "", ""
		f.Status, f.SubmittedAt, f.SubmittedBy = models.StatusDraft, "", ""
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).Updates(updates).Error; err != nil {
		return models.Form{}, fmt.Errorf("decide: %w", err)
	}
	f.LastUpdated = now

	row, found, err := m.openApproval(ctx, id)
	if err != nil {
		return models.Form{}, fmt.Errorf("decide: %w", err)
	}
	if !found {
		row = gormstore.ApprovalToRow(models.Approval{FormID: id, RequestedBy: submittedBy, RequestedAt: now})
		if err := m.db.WithContext(ctx).Table(m.tables.Approvals).Create(&row).Error; err != nil {
			return models.Form{}, fmt.Errorf("decide: %w", err)
		}
	}
	err = m.db.WithContext(ctx).Table(m.tables.Approvals).Where("id = ?", row.ID).
		Updates(map[string]any{"decided_by": approverID, "decided_at": now, "decision": string(decision), "note": note}).Error
	if err != nil {
		return models.Form{}, fmt.Errorf("decide: %w", err)
	}
	return f, nil
}

// Approve decides a submitted Form's open approval as approved.
func (m *formManager) Approve(ctx context.Context, id, approverID, note string) (models.Form, error) {
	return m.decide(ctx, id, approverID, note, models.DecisionApproved)
}

// Refuse decides a submitted Form's open approval as refused. A note is
// required — unlike Approve, a refusal must say why.
func (m *formManager) Refuse(ctx context.Context, id, approverID, note string) (models.Form, error) {
	if strings.TrimSpace(note) == "" {
		return models.Form{}, ErrMissingNote
	}
	return m.decide(ctx, id, approverID, note, models.DecisionRefused)
}

// ListAwaitingApproval returns every submitted Form, SubmittedAt-then-id
// order (a form with no SubmittedAt sorts last), truncated to limit when
// limit > 0.
func (m *formManager) ListAwaitingApproval(ctx context.Context, limit int) ([]models.Form, error) {
	var rows []gormstore.FormRow
	if err := m.db.WithContext(ctx).Table(m.tables.Forms).Where("status = ?", string(models.StatusSubmitted)).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListAwaitingApproval: %w", err)
	}
	out := make([]models.Form, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.FormFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SubmittedAt != out[j].SubmittedAt {
			if out[i].SubmittedAt == "" {
				return false
			}
			if out[j].SubmittedAt == "" {
				return true
			}
			return out[i].SubmittedAt < out[j].SubmittedAt
		}
		return out[i].ID < out[j].ID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListApprovals returns every approval cycle a Form has been through,
// newest first (RequestedAt-then-id descending).
func (m *formManager) ListApprovals(ctx context.Context, formID string) ([]models.Approval, error) {
	var rows []gormstore.ApprovalRow
	if err := m.db.WithContext(ctx).Table(m.tables.Approvals).Where("form_id = ?", formID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListApprovals: %w", err)
	}
	out := make([]models.Approval, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.ApprovalFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RequestedAt != out[j].RequestedAt {
			return out[i].RequestedAt > out[j].RequestedAt
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}
