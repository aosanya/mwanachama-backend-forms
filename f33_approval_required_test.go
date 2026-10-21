package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func newAuthoredDraft(t *testing.T, um mwanachamaforms.FormManager, author string) mwanachamaforms.Form {
	t.Helper()
	f, err := um.Create(context.Background(), mwanachamaforms.Form{
		Title: "Ward survey", OriginatorChapterID: "chapter-1",
		ClosesAt: futureRFC3339(24 * time.Hour), CreatedBy: author,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return f
}

func TestApprovalRequiredRefusesPublishingAnUnclearedDraft(t *testing.T) {
	um := newTestManager(t)
	um.UseApprovalRequired(true)
	f := newAuthoredDraft(t, um, "author")
	if _, _, err := um.Publish(context.Background(), f.ID, "", "author", ""); !errors.Is(err, mwanachamaforms.ErrNotApproved) {
		t.Fatalf("err = %v, want ErrNotApproved", err)
	}
	got, err := um.Get(context.Background(), f.ID)
	if err != nil || got.Status != mwanachamaforms.StatusDraft {
		t.Fatalf("form after refused publish = %+v, %v; want draft", got, err)
	}
}

func TestApprovalRequiredPublishesAFormClearedBySomebodyElse(t *testing.T) {
	um := newTestManager(t)
	um.UseApprovalRequired(true)
	f := newAuthoredDraft(t, um, "author")
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := um.Approve(context.Background(), f.ID, "reviewer", ""); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	out, _, err := um.Publish(context.Background(), f.ID, "", "author", "")
	if err != nil || out.Status != mwanachamaforms.StatusOpen {
		t.Fatalf("Publish = %+v, %v; want open", out, err)
	}
}

func TestApprovalRequiredRefusesTheAuthorClearingTheirOwnForm(t *testing.T) {
	for _, tc := range []struct{ name, creator, submitter string }{
		{"composer", "author", "someone-else"},
		{"submitter", "someone-else", "author"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			um := newTestManager(t)
			um.UseApprovalRequired(true)
			f := newAuthoredDraft(t, um, tc.creator)
			if _, err := um.Submit(context.Background(), f.ID, tc.submitter); err != nil {
				t.Fatalf("Submit: %v", err)
			}
			if _, err := um.Approve(context.Background(), f.ID, "author", ""); !errors.Is(err, mwanachamaforms.ErrSelfApproval) {
				t.Fatalf("err = %v, want ErrSelfApproval", err)
			}
			got, err := um.Get(context.Background(), f.ID)
			if err != nil || got.Status != mwanachamaforms.StatusSubmitted {
				t.Fatalf("form = %+v, %v; want still submitted", got, err)
			}
		})
	}
}

func TestApprovalRequiredRefusalNeedsAReasonAndReturnsTheFormToDraft(t *testing.T) {
	um := newTestManager(t)
	um.UseApprovalRequired(true)
	f := newAuthoredDraft(t, um, "author")
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := um.Refuse(context.Background(), f.ID, "reviewer", " "); !errors.Is(err, mwanachamaforms.ErrMissingNote) {
		t.Fatalf("err = %v, want ErrMissingNote", err)
	}
	out, err := um.Refuse(context.Background(), f.ID, "reviewer", "the closing date is wrong")
	if err != nil || out.Status != mwanachamaforms.StatusDraft {
		t.Fatalf("Refuse = %+v, %v; want draft", out, err)
	}
	approvals, err := um.ListApprovals(context.Background(), f.ID)
	if err != nil || len(approvals) != 1 || approvals[0].Note != "the closing date is wrong" {
		t.Fatalf("approvals = %+v, %v", approvals, err)
	}
}

func TestApprovalNotRequiredPublishesADraftAndLetsTheAuthorClear(t *testing.T) {
	um := newTestManager(t)
	f := newAuthoredDraft(t, um, "author")
	out, _, err := um.Publish(context.Background(), f.ID, "", "author", "")
	if err != nil || out.Status != mwanachamaforms.StatusOpen {
		t.Fatalf("Publish = %+v, %v; want open", out, err)
	}

	g := newAuthoredDraft(t, um, "author")
	if _, err := um.Submit(context.Background(), g.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := um.Approve(context.Background(), g.ID, "author", ""); err != nil {
		t.Fatalf("Approve: %v", err)
	}
}
