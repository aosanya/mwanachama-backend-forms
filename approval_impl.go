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

var errNoOpenApproval = errors.New("mwanachamaforms: no open approval")

// openApproval reads the form's undecided round, of which there is at most
// one — an empty decided_at is what makes a round open.
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

// decide is what Approve and Refuse share: it needs a form awaiting a
// decision, moves it on or back, and stamps the open round — synthesising
// one first in the case where none exists, which Submit does not produce but
// a database restored from elsewhere might.
func (m *formManager) decide(ctx context.Context, id, approverID, note string, decision models.Decision) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status != models.StatusSubmitted {
		return models.Form{}, ErrNotSubmitted
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

// Refuse needs a note where Approve does not: a refusal has to say why.
func (m *formManager) Refuse(ctx context.Context, id, approverID, note string) (models.Form, error) {
	if strings.TrimSpace(note) == "" {
		return models.Form{}, ErrMissingNote
	}
	return m.decide(ctx, id, approverID, note, models.DecisionRefused)
}

// ListAwaitingApproval returns every form waiting on a decision, the longest
// wait first. One with no submission stamp sorts last rather than first,
// where an empty string would otherwise put it.
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
