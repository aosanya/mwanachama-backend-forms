package mwanachamaforms_test

import (
	"context"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestAStandingFormKeepsEverySendAsItsOwnRecord(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Issues", Standing: true})
	if got, _ := mgr.Get(ctx, f.ID); !got.Standing {
		t.Fatalf("standing was not kept at compose: %+v", got)
	}
	for _, text := range []string{"broken pump", "leaking roof", "no light"} {
		if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: text}}); err != nil {
			t.Fatalf("SubmitAnswers %q: %v", text, err)
		}
	}
	sends, err := mgr.ListSubmissions(ctx, f.ID, "alice", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(sends) != 3 {
		t.Fatalf("sends = %d, want 3 kept apart: %+v", len(sends), sends)
	}
	keys := map[string]bool{}
	for _, s := range sends {
		if s.Answers != 1 || s.PeriodKey == "" || keys[s.PeriodKey] {
			t.Fatalf("send %+v is not its own record", s)
		}
		keys[s.PeriodKey] = true
	}
	answers, _ := mgr.ListAllAnswers(ctx, f.ID, "alice", "")
	given := map[string]bool{}
	for _, a := range answers {
		given[a.ValueText] = true
	}
	if len(answers) != 3 || !given["broken pump"] || !given["leaking roof"] || !given["no light"] {
		t.Fatalf("answers = %+v, want all three sends' answers kept", answers)
	}
}

func TestAnOrdinaryFormStillEditsItsOneSend(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Survey"})
	for _, text := range []string{"first", "second"} {
		if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: text}}); err != nil {
			t.Fatalf("SubmitAnswers: %v", err)
		}
	}
	sends, _ := mgr.ListSubmissions(ctx, f.ID, "alice", "")
	answers, _ := mgr.ListAnswers(ctx, f.ID, "alice")
	if len(sends) != 1 || sends[0].PeriodKey != "" || len(answers) != 1 || answers[0].ValueText != "second" {
		t.Fatalf("sends = %+v answers = %+v, want one send edited to the second", sends, answers)
	}
}

func TestStandingIsChosenAtComposeAndKeptThroughAnEdit(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, _ := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Issues", Standing: true})
	cur, _ := mgr.Get(ctx, f.ID)
	if cur.Status != mwanachamaforms.StatusDraft {
		t.Skip("form is not a draft after compose")
	}
	cur.Standing = false
	if _, err := mgr.Update(ctx, cur); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, _ := mgr.Get(ctx, f.ID); !got.Standing {
		t.Fatalf("an edit cleared standing: %+v", got)
	}
}

func TestEachSendOfAStandingFormIsListedAmongWhatAMemberAnswered(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Issues", Standing: true})
	for range 2 {
		if _, err := mgr.SubmitAnswers(ctx, f.ID, "alice", []mwanachamaforms.Answer{{QuestionID: q.ID, ValueText: "again"}}); err != nil {
			t.Fatalf("SubmitAnswers: %v", err)
		}
	}
	rows, err := mgr.ListAnswered(ctx, "alice")
	if err != nil {
		t.Fatalf("ListAnswered: %v", err)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.FormID == f.ID {
			seen[r.PeriodKey] = true
		}
	}
	if len(seen) != 2 {
		t.Fatalf("answered lists %d sends of the standing form, want 2: %+v", len(seen), rows)
	}
}
