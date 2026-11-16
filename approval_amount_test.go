package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func limitsOf(cap int) forms.ApprovalLimits {
	return func(_ context.Context, approverID string, amount int) error {
		if approverID == "approver-1" && amount <= cap {
			return nil
		}
		return forms.ErrOverApprovalLimit
	}
}

func submittedWithAmount(t *testing.T, m forms.FormManager, amount int, roles []string) models.Form {
	t.Helper()
	ctx := context.Background()
	m.UseApprovalRequired(true)
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic grant", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
		Amount: amount, ApprovalRoles: roles,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := m.Submit(ctx, f.ID, "author-1"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return f
}

func TestApprovalRefusesAnAmountOverTheApproversLimit(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsOf(100))
	f := submittedWithAmount(t, m, 101, nil)
	ctx := context.Background()
	if _, err := m.Approve(ctx, f.ID, "approver-1", ""); !errors.Is(err, forms.ErrOverApprovalLimit) {
		t.Fatalf("Approve over the limit = %v, want ErrOverApprovalLimit", err)
	}
	got, _ := m.Get(ctx, f.ID)
	if got.Status != models.StatusSubmitted {
		t.Fatalf("status = %s, want submitted", got.Status)
	}
	m.UseApprovalLimits(limitsOf(101))
	if _, err := m.Approve(ctx, f.ID, "approver-1", ""); err != nil {
		t.Fatalf("Approve at the limit = %v", err)
	}
}

func TestApprovalOfAnAmountWithoutALimitSourceIsRefused(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 5, nil)
	if _, err := m.Approve(context.Background(), f.ID, "approver-1", ""); !errors.Is(err, forms.ErrOverApprovalLimit) {
		t.Fatalf("Approve = %v, want ErrOverApprovalLimit", err)
	}
}

func TestAFormWithNoAmountNeedsNoSeat(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, nil)
	if _, err := m.Approve(context.Background(), f.ID, "approver-1", ""); err != nil {
		t.Fatalf("Approve = %v", err)
	}
}

func TestSlotApprovalRefusesAnAmountOverTheApproversLimit(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsOf(100))
	f := submittedWithAmount(t, m, 500, []string{"treasurer"})
	if _, err := m.ApproveSlot(context.Background(), f.ID, "approver-1", "treasurer", ""); !errors.Is(err, forms.ErrOverApprovalLimit) {
		t.Fatalf("ApproveSlot = %v, want ErrOverApprovalLimit", err)
	}
}

func TestANegativeAmountIsRefused(t *testing.T) {
	m := gatedManager(t, nil)
	_, err := m.Create(context.Background(), models.Form{
		Title: "Synthetic", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf, Amount: -1,
	})
	if !errors.Is(err, forms.ErrInvalidAmount) {
		t.Fatalf("Create = %v, want ErrInvalidAmount", err)
	}
}
