package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func sendOnce(t *testing.T, mgr mwanachamaforms.FormManager, formID, questionID, member, text string) {
	t.Helper()
	if _, err := mgr.SubmitAnswers(context.Background(), formID, member, []mwanachamaforms.Answer{{QuestionID: questionID, ValueText: text}}); err != nil {
		t.Fatalf("send as %s: %v", member, err)
	}
}

func TestAFreshSendIsNotPaid(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Seed order"})
	sendOnce(t, mgr, f.ID, q.ID, "alice", "10 bags")

	s, err := mgr.GetSubmission(context.Background(), f.ID, "alice")
	if err != nil || s.Paid || s.PaidAt != "" || s.PaidBy != "" {
		t.Fatalf("submission = %+v, %v, want not paid", s, err)
	}
}

func TestTheOrganiserMarksASendPaidAndTheListShowsIt(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Seed order"})
	sendOnce(t, mgr, f.ID, q.ID, "alice", "10 bags")
	sendOnce(t, mgr, f.ID, q.ID, "bob", "4 bags")

	marked, err := mgr.MarkPaid(ctx, f.ID, "alice", true, "coordinator")
	if err != nil || !marked.Paid || marked.PaidBy != "coordinator" || marked.PaidAt == "" {
		t.Fatalf("MarkPaid = %+v, %v, want paid by coordinator", marked, err)
	}
	mine, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || !mine.Paid {
		t.Fatalf("alice's own entry = %+v, %v, want paid", mine, err)
	}
	all, err := mgr.ListSubmissions(ctx, f.ID, "", "")
	if err != nil || len(all) != 2 {
		t.Fatalf("list = %+v, %v, want two", all, err)
	}
	paid := map[string]bool{}
	for _, s := range all {
		paid[s.MemberID] = s.Paid
	}
	if !paid["alice"] || paid["bob"] {
		t.Fatalf("paid by member = %v, want alice only", paid)
	}
}

func TestMarkingNotPaidClearsTheMark(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Seed order"})
	sendOnce(t, mgr, f.ID, q.ID, "alice", "10 bags")
	if _, err := mgr.MarkPaid(ctx, f.ID, "alice", true, "coordinator"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.MarkPaid(ctx, f.ID, "alice", false, "coordinator"); err != nil {
		t.Fatal(err)
	}
	s, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || s.Paid || s.PaidAt != "" || s.PaidBy != "" {
		t.Fatalf("submission = %+v, %v, want not paid and no marker", s, err)
	}
}

func TestAResendBeforeCloseKeepsThePaidMark(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Seed order"})
	sendOnce(t, mgr, f.ID, q.ID, "alice", "10 bags")
	marked, err := mgr.MarkPaid(ctx, f.ID, "alice", true, "coordinator")
	if err != nil {
		t.Fatal(err)
	}

	sendOnce(t, mgr, f.ID, q.ID, "alice", "12 bags")

	s, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil || !s.Paid || s.PaidAt != marked.PaidAt || s.PaidBy != "coordinator" || s.EditedAt == "" {
		t.Fatalf("after the resend = %+v, %v, want edited and still paid at %s", s, err, marked.PaidAt)
	}
}

func TestMarkingSomeoneWhoSentNothingIsNotFound(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	f, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Seed order"})

	_, err := mgr.MarkPaid(context.Background(), f.ID, "carol", true, "coordinator")
	if !errors.Is(err, mwanachamaforms.ErrSubmissionNotFound) {
		t.Fatalf("err = %v, want ErrSubmissionNotFound", err)
	}
}
