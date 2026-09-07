package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestPickUp(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	p, err := um.PickUp(context.Background(), f.ID, "chapter-2")
	if err != nil {
		t.Fatalf("PickUp: %v", err)
	}
	if p.State != mwanachamaforms.PropagationPickedUp || p.PickedUpAt == "" {
		t.Fatalf("unexpected propagation: %+v", p)
	}

	rollup, err := um.Rollup(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("Rollup: %v", err)
	}
	if rollup.PickedUp != 1 || rollup.Stalled != 0 {
		t.Fatalf("unexpected rollup: %+v", rollup)
	}

	// Idempotent repeat.
	again, err := um.PickUp(context.Background(), f.ID, "chapter-2")
	if err != nil {
		t.Fatalf("PickUp (repeat): %v", err)
	}
	if again.PickedUpAt != p.PickedUpAt {
		t.Fatalf("repeat pick-up changed timestamp: %+v", again)
	}
}

func TestPickUp_NotTargeted(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, err := um.PickUp(context.Background(), f.ID, "chapter-2"); !errors.Is(err, mwanachamaforms.ErrTargetNotFound) {
		t.Fatalf("err = %v, want ErrTargetNotFound", err)
	}
}
