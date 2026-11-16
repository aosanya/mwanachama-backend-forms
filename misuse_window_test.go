package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestClosedRound_ResidentCannotAnswerAfterClose(t *testing.T) {
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	ctx := context.Background()

	if err := r.sendAs("mem-1", "g-1", "while open"); err != nil {
		t.Fatalf("send while open: %v", err)
	}

	if _, err := r.m.Close(ctx, r.form.ID, "author-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := r.sendAs("mem-2", "g-1", "after close")
	if !errors.Is(err, forms.ErrFormNotOpen) {
		t.Fatalf("send after close: want ErrFormNotOpen, got %v", err)
	}
}
