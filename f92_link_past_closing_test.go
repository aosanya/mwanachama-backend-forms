package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestALinkPastItsClosingDateAnswersAsAWrongKeyDoes(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f := newOpenPublicForm(t, mgr, "late-key")
	moveClosesAtIntoThePast(t, db, f.ID)

	_, _, found, err := mgr.ResolveLinkKey(ctx, "late-key")
	if err != nil {
		t.Fatalf("ResolveLinkKey: %v", err)
	}
	if found {
		t.Fatal("a link past its closing date resolved")
	}

	_, lateErr := mgr.OpenPublicLink(ctx, "late-key")
	_, wrongErr := mgr.OpenPublicLink(ctx, "no-such-key")
	if !errors.Is(lateErr, mwanachamaforms.ErrLinkNotFound) || lateErr.Error() != wrongErr.Error() {
		t.Fatalf("late link err = %v, wrong key err = %v; want the same not-found", lateErr, wrongErr)
	}
	if _, err := mgr.PublicLinkQuestions(ctx, "late-key"); err == nil || err.Error() != wrongErr.Error() {
		t.Fatalf("questions of a late link err = %v, want %v", err, wrongErr)
	}
}

func TestALinkBeforeItsClosingDateStillOpens(t *testing.T) {
	mgr := newTestManager(t)
	newOpenPublicForm(t, mgr, "on-time-key")
	if _, err := mgr.OpenPublicLink(context.Background(), "on-time-key"); err != nil {
		t.Fatalf("OpenPublicLink: %v", err)
	}
}

func TestAnAdvisoryFormsLinkOpensPastItsClosingDate(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{
		Title: "Census", OriginatorChapterID: "chapter-1", ClosesAt: futureRFC3339(time.Hour),
		Audience: mwanachamaforms.AudiencePublic, WindowAdvisory: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := mgr.Publish(ctx, f.ID, "advisory-key", "admin", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	moveClosesAtIntoThePast(t, db, f.ID)
	if _, err := mgr.OpenPublicLink(ctx, "advisory-key"); err != nil {
		t.Fatalf("OpenPublicLink: %v", err)
	}
}
