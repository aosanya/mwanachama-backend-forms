package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestSubmit(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	out, err := um.Submit(context.Background(), f.ID, "author")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if out.Status != mwanachamaforms.StatusSubmitted || out.SubmittedBy != "author" {
		t.Fatalf("unexpected form: %+v", out)
	}
	approvals, err := um.ListApprovals(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListApprovals: %v", err)
	}
	if len(approvals) != 1 || approvals[0].DecidedAt != "" {
		t.Fatalf("unexpected approvals: %+v", approvals)
	}
}

func TestSubmit_NotDraft(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := um.Submit(context.Background(), f.ID, "author"); !errors.Is(err, mwanachamaforms.ErrNotDraft) {
		t.Fatalf("err = %v, want ErrNotDraft", err)
	}
}

func TestApprove(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	out, err := um.Approve(context.Background(), f.ID, "reviewer", "")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if out.Status != mwanachamaforms.StatusApproved || out.ApprovedBy != "reviewer" {
		t.Fatalf("unexpected form: %+v", out)
	}
	approvals, err := um.ListApprovals(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListApprovals: %v", err)
	}
	if len(approvals) != 1 || approvals[0].Decision != mwanachamaforms.DecisionApproved {
		t.Fatalf("unexpected approvals: %+v", approvals)
	}
}

func TestRefuse_RequiresNote(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := um.Refuse(context.Background(), f.ID, "reviewer", "   "); !errors.Is(err, mwanachamaforms.ErrMissingNote) {
		t.Fatalf("err = %v, want ErrMissingNote", err)
	}
}

func TestRefuse(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	out, err := um.Refuse(context.Background(), f.ID, "reviewer", "not ready")
	if err != nil {
		t.Fatalf("Refuse: %v", err)
	}
	if out.Status != mwanachamaforms.StatusDraft || out.SubmittedBy != "" {
		t.Fatalf("unexpected form: %+v", out)
	}
}

func TestWithdraw(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	out, err := um.Withdraw(context.Background(), f.ID, "author")
	if err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if out.Status != mwanachamaforms.StatusDraft {
		t.Fatalf("unexpected form: %+v", out)
	}
	approvals, err := um.ListApprovals(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListApprovals: %v", err)
	}
	if len(approvals) != 1 || approvals[0].Decision != mwanachamaforms.DecisionRefused {
		t.Fatalf("unexpected approvals: %+v", approvals)
	}
}

func TestWithdraw_NotWithdrawable(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Withdraw(context.Background(), f.ID, "author"); !errors.Is(err, mwanachamaforms.ErrNotWithdrawable) {
		t.Fatalf("err = %v, want ErrNotWithdrawable", err)
	}
}

func TestListAwaitingApproval(t *testing.T) {
	um := newTestManager(t)
	f1 := newDraftForm(t, um)
	f2 := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f1.ID, "author"); err != nil {
		t.Fatalf("Submit f1: %v", err)
	}
	if _, err := um.Submit(context.Background(), f2.ID, "author"); err != nil {
		t.Fatalf("Submit f2: %v", err)
	}
	out, err := um.ListAwaitingApproval(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListAwaitingApproval: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d forms, want 2", len(out))
	}
}
