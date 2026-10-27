package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestALockedFormTakesTheFirstSendAndRefusesAResend(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Pledge", LockedAfterSend: true})

	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "yes"}}); err != nil {
		t.Fatalf("first send: %v", err)
	}
	_, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "no"}})
	if !errors.Is(err, mwanachamaforms.ErrLockedAfterSend) {
		t.Fatalf("resend: err = %v, want ErrLockedAfterSend", err)
	}
	got, err := mgr.ListAnswers(ctx, f.ID, "alice")
	if err != nil || len(got) != 1 || got[0].ValueText != "yes" || got[0].EditedAt != "" {
		t.Fatalf("after the refused resend = %+v, %v, want the answer as first sent", got, err)
	}
}

func TestALockedFormStillTakesAFirstSendFromAnotherMember(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Pledge", LockedAfterSend: true})

	for _, who := range []string{"alice", "bob"} {
		if _, err := mgr.SubmitAnswers(ctx, f.ID, who, []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "yes"}}); err != nil {
			t.Fatalf("first send by %s: %v", who, err)
		}
	}
}

func TestADefaultFormIsStillEditableAfterASend(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Ward check"})
	if f.LockedAfterSend {
		t.Fatal("a form composed without the flag is locked")
	}
	for _, v := range []string{"fine", "better"} {
		if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: v}}); err != nil {
			t.Fatalf("send %q: %v", v, err)
		}
	}
}

func TestTheLockIsSetAtCreateAndShownOnTheAnsweredList(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Pledge", LockedAfterSend: true})
	if got, err := mgr.Get(ctx, f.ID); err != nil || !got.LockedAfterSend {
		t.Fatalf("Get = %+v, %v, want locked_after_send", got, err)
	}
	if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "yes"}}); err != nil {
		t.Fatalf("send: %v", err)
	}
	list, err := mgr.ListAnswered(ctx, "alice")
	if err != nil || len(list) != 1 || !list[0].LockedAfterSend {
		t.Fatalf("answered = %+v, %v, want one locked entry", list, err)
	}
}
