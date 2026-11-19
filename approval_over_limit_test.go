package mwanachamaforms_test

import (
	"context"
	"errors"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"testing"
)

func limitsThatFlag(cap int) forms.ApprovalLimits {
	return func(_ context.Context, approverID string, amount int) (forms.ApprovalCover, error) {
		if approverID != "mayor-1" {
			return forms.ApprovalCover{}, forms.ErrOverApprovalLimit
		}
		if amount <= cap {
			return forms.ApprovalCover{}, nil
		}
		return forms.ApprovalCover{OverLimit: true, Limit: cap}, nil
	}
}
func approvalRowOf(t *testing.T, m forms.FormManager, formID string) models.Approval {
	t.Helper()
	rounds, err := m.ListApprovals(context.Background(), formID)
	if err != nil || len(rounds) == 0 {
		t.Fatalf("ListApprovals = %v %v, want a round", rounds, err)
	}
	return rounds[0]
}
func TestAnApprovalAboveTheLimitIsStoredWithTheFlagAndTheLimitThatApplied(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsThatFlag(500_000))
	f := submittedWithAmount(t, m, 900_000, nil)
	ctx := context.Background()
	approved, err := m.Approve(ctx, f.ID, "mayor-1", "")
	if err != nil {
		t.Fatalf("Approve above the limit = %v, want it accepted", err)
	}
	if approved.Status != models.StatusApproved || !approved.OverLimit || approved.LimitInForce != 500_000 {
		t.Fatalf("approved form = status %s over %v limit %d, want approved, over, 500000", approved.Status, approved.OverLimit, approved.LimitInForce)
	}
	read, _ := m.Get(ctx, f.ID)
	if !read.OverLimit || read.LimitInForce != 500_000 {
		t.Fatalf("read back = over %v limit %d, want the flag kept", read.OverLimit, read.LimitInForce)
	}
	round := approvalRowOf(t, m, f.ID)
	if !round.OverLimit || round.LimitInForce != 500_000 {
		t.Fatalf("approval row = over %v limit %d, want the flag and the limit", round.OverLimit, round.LimitInForce)
	}
}
func TestAnApprovalWithinTheLimitCarriesNoFlag(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsThatFlag(500_000))
	f := submittedWithAmount(t, m, 400_000, nil)
	approved, err := m.Approve(context.Background(), f.ID, "mayor-1", "")
	if err != nil {
		t.Fatalf("Approve within the limit = %v", err)
	}
	if approved.OverLimit || approved.LimitInForce != 0 {
		t.Fatalf("approved form = over %v limit %d, want no flag", approved.OverLimit, approved.LimitInForce)
	}
	if round := approvalRowOf(t, m, f.ID); round.OverLimit || round.LimitInForce != 0 {
		t.Fatalf("approval row = over %v limit %d, want no flag", round.OverLimit, round.LimitInForce)
	}
}
func TestARefusedApproverStillStopsAtTheLimit(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsThatFlag(500_000))
	f := submittedWithAmount(t, m, 900_000, nil)
	if _, err := m.Approve(context.Background(), f.ID, "clerk-1", ""); !errors.Is(err, forms.ErrOverApprovalLimit) {
		t.Fatalf("Approve by a role that may not = %v, want ErrOverApprovalLimit", err)
	}
}
func TestASlotGrantedAboveTheLimitFlagsTheFormOnceItClears(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsThatFlag(500_000))
	f := submittedWithAmount(t, m, 900_000, []string{"mayor"})
	ctx := context.Background()
	cleared, err := m.ApproveSlot(ctx, f.ID, "mayor-1", "mayor", "")
	if err != nil {
		t.Fatalf("ApproveSlot above the limit = %v", err)
	}
	if cleared.Status != models.StatusApproved || !cleared.OverLimit || cleared.LimitInForce != 500_000 {
		t.Fatalf("cleared form = status %s over %v limit %d", cleared.Status, cleared.OverLimit, cleared.LimitInForce)
	}
}
func TestAResentFormStartsWithoutTheEarlierFlag(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseApprovalLimits(limitsThatFlag(500_000))
	f := submittedWithAmount(t, m, 900_000, nil)
	ctx := context.Background()
	if _, err := m.Approve(ctx, f.ID, "mayor-1", ""); err != nil {
		t.Fatalf("Approve = %v", err)
	}
	withdrawn, err := m.Withdraw(ctx, f.ID, "author-1")
	if err != nil {
		t.Fatalf("Withdraw = %v", err)
	}
	if withdrawn.OverLimit || withdrawn.LimitInForce != 0 {
		t.Fatalf("withdrawn form = over %v limit %d, want the flag cleared", withdrawn.OverLimit, withdrawn.LimitInForce)
	}
}
