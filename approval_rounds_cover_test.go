package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestRefusingNeedsANoteAndLeavesARefusedRoundOnTheRecord(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, nil)
	ctx := context.Background()

	if _, err := m.Refuse(ctx, f.ID, "approver-1", "   "); !errors.Is(err, forms.ErrMissingNote) {
		t.Fatalf("Refuse without a note = %v, want ErrMissingNote", err)
	}
	if got, _ := m.Get(ctx, f.ID); got.Status != models.StatusSubmitted {
		t.Fatalf("a refused attempt without a note changed the status to %s", got.Status)
	}

	refused, err := m.Refuse(ctx, f.ID, "approver-1", "The budget line is missing")
	if err != nil {
		t.Fatalf("Refuse: %v", err)
	}
	if refused.Status == models.StatusSubmitted {
		t.Fatal("a refused form still waits for approval")
	}

	rounds, err := m.ListApprovals(ctx, f.ID)
	if err != nil || len(rounds) != 1 {
		t.Fatalf("ListApprovals = %d rounds, %v, want 1", len(rounds), err)
	}
	if rounds[0].Decision != models.DecisionRefused || rounds[0].Note != "The budget line is missing" || rounds[0].DecidedBy != "approver-1" {
		t.Fatalf("the round does not say who refused it and why: %+v", rounds[0])
	}

	if _, err := m.Refuse(ctx, f.ID, "approver-1", "again"); !errors.Is(err, forms.ErrNotSubmitted) {
		t.Fatalf("refusing a form that is not waiting = %v, want ErrNotSubmitted", err)
	}
}

func TestWithdrawingASubmittedFormReturnsItToDraftAndClosesItsRound(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, nil)
	ctx := context.Background()

	back, err := m.Withdraw(ctx, f.ID, "author-1")
	if err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if back.Status != models.StatusDraft || back.SubmittedAt != "" || back.SubmittedBy != "" {
		t.Fatalf("a withdrawn form must be a clean draft: %+v", back)
	}

	rounds, err := m.ListApprovals(ctx, f.ID)
	if err != nil || len(rounds) != 1 {
		t.Fatalf("ListApprovals = %d rounds, %v, want 1", len(rounds), err)
	}
	if rounds[0].Decision != models.DecisionRefused || rounds[0].DecidedBy != "author-1" || rounds[0].Note == "" {
		t.Fatalf("the round must be closed as withdrawn by its author: %+v", rounds[0])
	}

	if _, err := m.Withdraw(ctx, f.ID, "author-1"); !errors.Is(err, forms.ErrNotWithdrawable) {
		t.Fatalf("withdrawing a draft = %v, want ErrNotWithdrawable", err)
	}
}

func TestSubmittingAgainAfterAWithdrawalOpensANewRoundAndListsNewestFirst(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, nil)
	ctx := context.Background()
	if _, err := m.Withdraw(ctx, f.ID, "author-1"); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if _, err := m.Submit(ctx, f.ID, "author-1"); err != nil {
		t.Fatalf("Submit again: %v", err)
	}
	rounds, err := m.ListApprovals(ctx, f.ID)
	if err != nil || len(rounds) != 2 {
		t.Fatalf("ListApprovals = %d rounds, %v, want 2", len(rounds), err)
	}
	if rounds[0].DecidedAt != "" || rounds[1].DecidedAt == "" {
		t.Fatalf("the open round must come first and the closed one after: %+v", rounds)
	}
}

func TestSubmittingNeedsApprovalToBeRequiredAndADraft(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	f := submittedWithAmount(t, m, 0, nil)
	if _, err := m.Submit(ctx, f.ID, "author-1"); !errors.Is(err, forms.ErrNotDraft) {
		t.Fatalf("submitting twice = %v, want ErrNotDraft", err)
	}
	m.UseApprovalRequired(false)
	if _, err := m.Submit(ctx, f.ID, "author-1"); !errors.Is(err, forms.ErrApprovalNotRequired) {
		t.Fatalf("submitting where nothing needs approval = %v, want ErrApprovalNotRequired", err)
	}
}

func TestTheApprovalQueueHoldsSubmittedFormsOldestFirstAndHonoursALimit(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	first := submittedWithAmount(t, m, 0, nil)
	second := submittedWithAmount(t, m, 0, nil)
	draft, err := m.Create(ctx, models.Form{
		Title: "Synthetic draft", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	queue, err := m.ListAwaitingApproval(ctx, 0)
	if err != nil || len(queue) != 2 {
		t.Fatalf("queue = %d forms, %v, want the two submitted ones", len(queue), err)
	}
	for _, f := range queue {
		if f.ID == draft.ID {
			t.Fatal("a draft is not waiting for approval")
		}
	}
	if queue[0].SubmittedAt > queue[1].SubmittedAt {
		t.Fatalf("the queue must run oldest first: %s then %s", queue[0].SubmittedAt, queue[1].SubmittedAt)
	}

	limited, err := m.ListAwaitingApproval(ctx, 1)
	if err != nil || len(limited) != 1 || limited[0].ID != queue[0].ID {
		t.Fatalf("a limit of one must give the oldest: %+v, %v (first %s, second %s)", limited, err, first.ID, second.ID)
	}
}

func TestEachSlotOfAMultiSlotFormIsGrantedByADifferentApprover(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, []string{"finance", "legal"})
	ctx := context.Background()

	if _, err := m.Approve(ctx, f.ID, "approver-1", ""); !errors.Is(err, forms.ErrSlotRequired) {
		t.Fatalf("approving without naming a slot = %v, want ErrSlotRequired", err)
	}
	if _, err := m.ApproveSlot(ctx, f.ID, "approver-1", "audit", ""); !errors.Is(err, forms.ErrUnknownSlot) {
		t.Fatalf("approving a slot the form does not declare = %v, want ErrUnknownSlot", err)
	}
	if _, err := m.ApproveSlot(ctx, f.ID, "approver-1", "finance", ""); err != nil {
		t.Fatalf("ApproveSlot finance: %v", err)
	}
	if _, err := m.ApproveSlot(ctx, f.ID, "approver-1", "finance", ""); !errors.Is(err, forms.ErrSlotGranted) && !errors.Is(err, forms.ErrApproverHoldsSlot) {
		t.Fatalf("granting the same slot twice = %v, want ErrSlotGranted or ErrApproverHoldsSlot", err)
	}
	if _, err := m.ApproveSlot(ctx, f.ID, "approver-1", "legal", ""); !errors.Is(err, forms.ErrApproverHoldsSlot) {
		t.Fatalf("one approver granting a second slot = %v, want ErrApproverHoldsSlot", err)
	}
	if got, _ := m.Get(ctx, f.ID); got.Status != models.StatusSubmitted {
		t.Fatalf("one slot of two must leave the form waiting, status = %s", got.Status)
	}

	cleared, err := m.ApproveSlot(ctx, f.ID, "approver-2", "legal", "")
	if err != nil {
		t.Fatalf("ApproveSlot legal: %v", err)
	}
	if cleared.Status != models.StatusApproved {
		t.Fatalf("both slots granted must clear the form, status = %s", cleared.Status)
	}
}
