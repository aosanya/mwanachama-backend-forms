package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type raceRound struct {
	m        forms.FormManager
	db       *gorm.DB
	s        *spec.Spec
	form     models.Form
	question models.Question
}

func openRaceRound(t *testing.T, locked bool) raceRound {
	t.Helper()
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic consultation", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
		LockedAfterSend: locked,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Your view", AnswerType: models.AnswerFreeText}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := m.AddTarget(ctx, models.Target{FormID: f.ID, ChapterID: "g-1"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := m.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return raceRound{m: m, db: db, s: s, form: f, question: q}
}

func (r raceRound) send(member, view string) error {
	_, err := r.m.SubmitAnswers(context.Background(), r.form.ID, member, []models.Answer{{QuestionID: r.question.ID, ValueText: view}})
	return err
}

func (r raceRound) tableOf(t *testing.T, role string) string {
	t.Helper()
	o, ok := r.s.Object(role)
	if !ok {
		t.Fatalf("no %s object", role)
	}
	return r.s.TableFor(o)
}

func (r raceRound) firstAttemptSeesAnEmptyWorld(t *testing.T) {
	t.Helper()
	r.firstAttemptSees(t, true, true)
}

func (r raceRound) onlyTheLockCheckIsStale(t *testing.T) {
	t.Helper()
	r.firstAttemptSees(t, false, true)
}

func (r raceRound) firstAttemptSees(t *testing.T, blankRows, blankCounts bool) {
	t.Helper()
	tables := map[string]bool{r.tableOf(t, "answer"): true, r.tableOf(t, "submission"): true}
	armed := true
	err := r.db.Callback().Query().After("gorm:query").Register("stale:"+t.Name(), func(tx *gorm.DB) {
		if !armed || !tables[tx.Statement.Table] {
			return
		}
		switch dest := tx.Statement.Dest.(type) {
		case *[]map[string]any:
			if blankRows {
				*dest = nil
			}
		case *int64:
			if blankCounts {
				*dest = 0
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	err = r.db.Callback().Create().After("gorm:create").Register("disarm:"+t.Name(), func(tx *gorm.DB) {
		if tables[tx.Statement.Table] {
			armed = false
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (r raceRound) stored(t *testing.T, member string) (answers []models.Answer, submissions []models.Submission) {
	t.Helper()
	ctx := context.Background()
	answers, err := r.m.ListAllAnswers(ctx, r.form.ID, member, "")
	if err != nil {
		t.Fatalf("ListAllAnswers: %v", err)
	}
	submissions, err = r.m.ListSubmissions(ctx, r.form.ID, member, "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	return answers, submissions
}

func TestALosingRaceOnASendReplacesTheWinnerNotAStoreError(t *testing.T) {
	r := openRaceRound(t, false)
	if err := r.send("mem-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	r.firstAttemptSeesAnEmptyWorld(t)
	if err := r.send("mem-1", "close the market"); err != nil {
		t.Fatalf("a send that lost the race to the same member's send = %v, want it to replace the winner", err)
	}
	answers, submissions := r.stored(t, "mem-1")
	if len(answers) != 1 || answers[0].ValueText != "close the market" || len(submissions) != 1 {
		t.Fatalf("stored %+v and %+v, want the later answer alone under one send", answers, submissions)
	}
}

func TestALosingRaceOnALockedSendIsRefusedNotAStoreError(t *testing.T) {
	r := openRaceRound(t, true)
	if err := r.send("mem-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	r.firstAttemptSeesAnEmptyWorld(t)
	if err := r.send("mem-1", "close the market"); !errors.Is(err, forms.ErrLockedAfterSend) {
		t.Fatalf("a send that lost the race on a locked form = %v, want ErrLockedAfterSend", err)
	}
	answers, submissions := r.stored(t, "mem-1")
	if len(answers) != 1 || answers[0].ValueText != "keep the market open" || len(submissions) != 1 {
		t.Fatalf("stored %+v and %+v, want the winner alone", answers, submissions)
	}
}

func TestAnEditThatSlippedPastTheLockCheckOnALockedFormIsRefusedNotApplied(t *testing.T) {
	r := openRaceRound(t, true)
	if err := r.send("mem-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	r.onlyTheLockCheckIsStale(t)
	if err := r.send("mem-1", "close the market"); !errors.Is(err, forms.ErrLockedAfterSend) {
		t.Fatalf("an edit that found the winner's answer on a locked form = %v, want ErrLockedAfterSend", err)
	}
	answers, submissions := r.stored(t, "mem-1")
	if len(answers) != 1 || answers[0].ValueText != "keep the market open" || len(submissions) != 1 {
		t.Fatalf("stored %+v and %+v, want the winner alone", answers, submissions)
	}
}
