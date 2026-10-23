package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) UseApprovalRequired(required bool) { m.approvalRequired = required }

func (m *formManager) UseDefaultMaxFiles(n int) { m.defaultMaxFiles = n }

func (m *formManager) isAuthor(f models.Form, actorID string) bool {
	return actorID != "" && (actorID == f.CreatedBy || actorID == f.SubmittedBy)
}

var errNoOpenApproval = errors.New("mwanachamaforms: no open approval")

func (m *formManager) openApproval(ctx context.Context, formID string) (models.Approval, bool, error) {
	var a models.Approval
	q := m.q(ctx, roleApproval).Where("form_id = ? AND decided_at = ?", formID, "")
	err := m.take(q, roleApproval, &a, errNoOpenApproval)
	if errors.Is(err, errNoOpenApproval) {
		return models.Approval{}, false, nil
	}
	if err != nil {
		return models.Approval{}, false, err
	}
	return a, true, nil
}

func (m *formManager) Submit(ctx context.Context, id, submittedBy string) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusDraft {
		return models.Form{}, ErrNotDraft
	}

	now := models.NowRFC3339()
	f.Status, f.SubmittedAt, f.SubmittedBy, f.LastUpdated = models.StatusSubmitted, now, submittedBy, now
	round := models.Approval{
		ID: newID(), FormID: id, RequestedBy: submittedBy, RequestedAt: now,
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.replaceTx(tx, roleForm, id, f); err != nil {
			return err
		}
		return m.insertTx(tx, roleApproval, round)
	})
	if err != nil {
		return models.Form{}, fmt.Errorf("Submit: %w", err)
	}
	return f, nil
}

func (m *formManager) Withdraw(ctx context.Context, id, actorID string) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusSubmitted && f.Status != models.StatusApproved {
		return models.Form{}, ErrNotWithdrawable
	}

	open, found, err := m.openApproval(ctx, id)
	if err != nil {
		return models.Form{}, fmt.Errorf("Withdraw: %w", err)
	}

	now := models.NowRFC3339()
	f.Status = models.StatusDraft
	f.SubmittedAt, f.SubmittedBy = "", ""
	f.ApprovedAt, f.ApprovedBy = "", ""
	f.LastUpdated = now

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.replaceTx(tx, roleForm, id, f); err != nil {
			return err
		}
		if !found {
			return nil
		}
		open.DecidedBy, open.DecidedAt = actorID, now
		open.Decision = models.DecisionRefused
		open.Note = "withdrawn by its author before a decision"
		return m.replaceTx(tx, roleApproval, open.ID, open)
	})
	if err != nil {
		return models.Form{}, fmt.Errorf("Withdraw: %w", err)
	}
	return f, nil
}

func (m *formManager) decide(ctx context.Context, id, approverID, note string, decision models.Decision) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusSubmitted {
		return models.Form{}, ErrNotSubmitted
	}
	if m.approvalRequired && decision == models.DecisionApproved && m.isAuthor(f, approverID) {
		return models.Form{}, ErrSelfApproval
	}
	if decision == models.DecisionApproved && len(f.ApprovalRoles) > 0 {
		return models.Form{}, ErrSlotRequired
	}
	submittedBy := f.SubmittedBy

	round, found, err := m.openApproval(ctx, id)
	if err != nil {
		return models.Form{}, fmt.Errorf("decide: %w", err)
	}

	now := models.NowRFC3339()
	if decision == models.DecisionApproved {
		f.Status, f.ApprovedAt, f.ApprovedBy = models.StatusApproved, now, approverID
	} else {
		f.Status, f.SubmittedAt, f.SubmittedBy = models.StatusDraft, "", ""
	}
	f.LastUpdated = now

	round.DecidedBy, round.DecidedAt, round.Decision, round.Note = approverID, now, decision, note

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.replaceTx(tx, roleForm, id, f); err != nil {
			return err
		}
		if !found {
			round.ID = newID()
			round.FormID, round.RequestedBy, round.RequestedAt = id, submittedBy, now
			return m.insertTx(tx, roleApproval, round)
		}
		return m.replaceTx(tx, roleApproval, round.ID, round)
	})
	if err != nil {
		return models.Form{}, fmt.Errorf("decide: %w", err)
	}
	return f, nil
}

func (m *formManager) Approve(ctx context.Context, id, approverID, note string) (models.Form, error) {
	return m.decide(ctx, id, approverID, note, models.DecisionApproved)
}

func (m *formManager) ApproveSlot(ctx context.Context, id, approverID, slot, note string) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if len(f.ApprovalRoles) == 0 {
		return m.decide(ctx, id, approverID, note, models.DecisionApproved)
	}
	if f.Status != models.StatusSubmitted {
		return models.Form{}, ErrNotSubmitted
	}
	if m.approvalRequired && m.isAuthor(f, approverID) {
		return models.Form{}, ErrSelfApproval
	}
	if !declaresSlot(f.ApprovalRoles, slot) {
		return models.Form{}, ErrUnknownSlot
	}
	round, found, err := m.openApproval(ctx, id)
	if err != nil {
		return models.Form{}, fmt.Errorf("ApproveSlot: %w", err)
	}
	if !found {
		return models.Form{}, ErrNotSubmitted
	}
	grants, err := m.slotGrants(ctx, round)
	if err != nil {
		return models.Form{}, fmt.Errorf("ApproveSlot: %w", err)
	}
	for _, g := range grants {
		if g.Slot == slot {
			return models.Form{}, ErrSlotGranted
		}
		if g.DecidedBy == approverID {
			return models.Form{}, ErrApproverHoldsSlot
		}
	}

	now := models.NowRFC3339()
	grant := models.Approval{
		ID: newID(), FormID: id, RequestedBy: round.RequestedBy, RequestedAt: round.RequestedAt,
		DecidedBy: approverID, DecidedAt: now, Decision: models.DecisionApproved, Note: note, Slot: slot,
	}
	cleared := len(grants)+1 == len(distinctSlots(f.ApprovalRoles))
	if cleared {
		f.Status, f.ApprovedAt, f.ApprovedBy, f.LastUpdated = models.StatusApproved, now, approverID, now
		round.DecidedBy, round.DecidedAt, round.Decision = approverID, now, models.DecisionApproved
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.insertTx(tx, roleApproval, grant); err != nil {
			return err
		}
		if !cleared {
			return nil
		}
		if err := m.replaceTx(tx, roleForm, id, f); err != nil {
			return err
		}
		return m.replaceTx(tx, roleApproval, round.ID, round)
	})
	if err != nil {
		return models.Form{}, fmt.Errorf("ApproveSlot: %w", err)
	}
	return f, nil
}

func (m *formManager) slotGrants(ctx context.Context, round models.Approval) ([]models.Approval, error) {
	return listOf[models.Approval](m,
		m.q(ctx, roleApproval).Where("form_id = ? AND requested_at = ? AND slot <> ? AND decision = ?",
			round.FormID, round.RequestedAt, "", string(models.DecisionApproved)), roleApproval)
}

func declaresSlot(roles []string, slot string) bool {
	if strings.TrimSpace(slot) == "" {
		return false
	}
	for _, r := range roles {
		if r == slot {
			return true
		}
	}
	return false
}

func distinctSlots(roles []string) map[string]bool {
	out := map[string]bool{}
	for _, r := range roles {
		if strings.TrimSpace(r) != "" {
			out[r] = true
		}
	}
	return out
}

func (m *formManager) Refuse(ctx context.Context, id, approverID, note string) (models.Form, error) {
	if strings.TrimSpace(note) == "" {
		return models.Form{}, ErrMissingNote
	}
	return m.decide(ctx, id, approverID, note, models.DecisionRefused)
}

func (m *formManager) ListAwaitingApproval(ctx context.Context, limit int) ([]models.Form, error) {
	out, err := listOf[models.Form](m,
		m.q(ctx, roleForm).Where("status = ?", string(models.StatusSubmitted)), roleForm)
	if err != nil {
		return nil, fmt.Errorf("ListAwaitingApproval: %w", err)
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

func (m *formManager) ListApprovals(ctx context.Context, formID string) ([]models.Approval, error) {
	out, err := listOf[models.Approval](m,
		m.q(ctx, roleApproval).Where("form_id = ?", formID), roleApproval)
	if err != nil {
		return nil, fmt.Errorf("ListApprovals: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RequestedAt != out[j].RequestedAt {
			return out[i].RequestedAt > out[j].RequestedAt
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}
