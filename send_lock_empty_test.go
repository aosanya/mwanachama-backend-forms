package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

func (r raceRound) sendNothing(member string) error {
	_, err := r.m.SubmitAnswers(context.Background(), r.form.ID, member, nil)
	return err
}

func TestAnEmptySendThatSlippedPastTheLockCheckOnALockedFormIsRefusedNotApplied(t *testing.T) {
	r := openRaceRound(t, true)
	if err := r.sendNothing("mem-1"); err != nil {
		t.Fatalf("first empty send: %v", err)
	}
	_, before := r.stored(t, "mem-1")
	r.onlyTheLockCheckIsStale(t)
	if err := r.sendNothing("mem-1"); !errors.Is(err, forms.ErrLockedAfterSend) {
		t.Fatalf("an empty send that found the winner's send on a locked form = %v, want ErrLockedAfterSend", err)
	}
	_, after := r.stored(t, "mem-1")
	if len(after) != 1 || after[0].ID != before[0].ID || after[0].EditedAt != "" {
		t.Fatalf("stored %+v, want the winner's send untouched", after)
	}
}

func TestAnEmptySendThenASendWithAnswersThatSlippedPastTheLockCheckIsRefused(t *testing.T) {
	r := openRaceRound(t, true)
	if err := r.sendNothing("mem-1"); err != nil {
		t.Fatalf("first empty send: %v", err)
	}
	r.onlyTheLockCheckIsStale(t)
	if err := r.send("mem-1", "close the market"); !errors.Is(err, forms.ErrLockedAfterSend) {
		t.Fatalf("a send with answers that found the winner's empty send on a locked form = %v, want ErrLockedAfterSend", err)
	}
	answers, submissions := r.stored(t, "mem-1")
	if len(answers) != 0 || len(submissions) != 1 || submissions[0].EditedAt != "" {
		t.Fatalf("stored %+v and %+v, want the empty send alone", answers, submissions)
	}
}
