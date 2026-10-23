package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func newSlottedForm(t *testing.T, um mwanachamaforms.FormManager) mwanachamaforms.Form {
	t.Helper()
	f, err := um.Create(context.Background(), mwanachamaforms.Form{
		Title:               "Two reviews",
		OriginatorChapterID: "chapter-1",
		ClosesAt:            futureRFC3339(24 * time.Hour),
		ApprovalRoles:       []string{"lead", "privacy"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return f
}

func TestASlottedFormClearsOnlyOnceEverySlotIsGrantedByDifferentApprovers(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newSlottedForm(t, um)

	if _, err := um.Approve(ctx, f.ID, "r1", ""); !errors.Is(err, mwanachamaforms.ErrSlotRequired) {
		t.Fatalf("plain approve: err = %v, want ErrSlotRequired", err)
	}
	if _, err := um.ApproveSlot(ctx, f.ID, "r1", "other", ""); !errors.Is(err, mwanachamaforms.ErrUnknownSlot) {
		t.Fatalf("unknown slot: err = %v", err)
	}
	out, err := um.ApproveSlot(ctx, f.ID, "r1", "lead", "")
	if err != nil {
		t.Fatalf("ApproveSlot lead: %v", err)
	}
	if out.Status != mwanachamaforms.StatusSubmitted {
		t.Fatalf("status after one slot = %s, want submitted", out.Status)
	}
	if _, _, err := um.Publish(ctx, f.ID, "", "publisher", ""); !errors.Is(err, mwanachamaforms.ErrAwaitingApproval) {
		t.Fatalf("publish with one slot: err = %v, want ErrAwaitingApproval", err)
	}
	if _, err := um.ApproveSlot(ctx, f.ID, "r2", "lead", ""); !errors.Is(err, mwanachamaforms.ErrSlotGranted) {
		t.Fatalf("slot twice: err = %v", err)
	}
	if _, err := um.ApproveSlot(ctx, f.ID, "r1", "privacy", ""); !errors.Is(err, mwanachamaforms.ErrApproverHoldsSlot) {
		t.Fatalf("same approver two slots: err = %v", err)
	}
	out, err = um.ApproveSlot(ctx, f.ID, "r2", "privacy", "")
	if err != nil {
		t.Fatalf("ApproveSlot privacy: %v", err)
	}
	if out.Status != mwanachamaforms.StatusApproved {
		t.Fatalf("status after both slots = %s, want approved", out.Status)
	}
}

func TestARefusalDiscardsTheRoundsGrantedSlots(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newSlottedForm(t, um)
	if _, err := um.ApproveSlot(ctx, f.ID, "r1", "lead", ""); err != nil {
		t.Fatalf("ApproveSlot: %v", err)
	}
	if _, err := um.Refuse(ctx, f.ID, "r2", "no"); err != nil {
		t.Fatalf("Refuse: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := um.Submit(ctx, f.ID, "author"); err != nil {
		t.Fatalf("Submit again: %v", err)
	}
	out, err := um.ApproveSlot(ctx, f.ID, "r3", "lead", "")
	if err != nil {
		t.Fatalf("lead in a new round: %v", err)
	}
	if out.Status != mwanachamaforms.StatusSubmitted {
		t.Fatalf("status = %s", out.Status)
	}
}

func TestAFormWithoutSlotsTakesApproveSlotAsAPlainApproval(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(ctx, f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	out, err := um.ApproveSlot(ctx, f.ID, "r1", "", "")
	if err != nil || out.Status != mwanachamaforms.StatusApproved {
		t.Fatalf("out = %+v, err = %v", out.Status, err)
	}
}
