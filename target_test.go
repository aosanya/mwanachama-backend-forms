package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestAddTarget(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	tg, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"})
	if err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if tg.ID == "" || tg.ChapterID != "chapter-2" {
		t.Fatalf("unexpected target: %+v", tg)
	}
}

func TestAddTarget_Duplicate(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"}); !errors.Is(err, mwanachamaforms.ErrDuplicateTarget) {
		t.Fatalf("err = %v, want ErrDuplicateTarget", err)
	}
}

func TestAFormMayBeAimedAtSeveralGroups(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	for _, g := range []string{"chapter-2", "chapter-3"} {
		if _, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: g}); err != nil {
			t.Fatalf("AddTarget %s: %v", g, err)
		}
	}
	held, err := um.ListTargets(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(held) != 2 || held[0].ChapterID != "chapter-2" || held[1].ChapterID != "chapter-3" {
		t.Fatalf("targets = %+v, want chapter-2 and chapter-3", held)
	}
}

func TestRemovingTheOneTargetLetsTheFormBeAimedElsewhere(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	first, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"})
	if err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if err := um.RemoveTarget(context.Background(), f.ID, first.ID); err != nil {
		t.Fatalf("RemoveTarget: %v", err)
	}
	if _, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-3"}); err != nil {
		t.Fatalf("AddTarget after removing the first: %v", err)
	}
}

func TestAddTarget_PublicFormRefused(t *testing.T) {
	um := newTestManager(t)
	f, err := um.Create(context.Background(), mwanachamaforms.Form{
		Title: "Public", OriginatorChapterID: "chapter-1", ClosesAt: futureRFC3339(time.Hour), Audience: mwanachamaforms.AudiencePublic,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"}); !errors.Is(err, mwanachamaforms.ErrPublicFormNoTargets) {
		t.Fatalf("err = %v, want ErrPublicFormNoTargets", err)
	}
}

func TestRemoveTarget(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	tg, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f.ID, ChapterID: "chapter-2"})
	if err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if err := um.RemoveTarget(context.Background(), f.ID, tg.ID); err != nil {
		t.Fatalf("RemoveTarget: %v", err)
	}
	out, err := um.ListTargets(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no targets, got %+v", out)
	}
}

func TestRemoveTarget_ForeignTarget(t *testing.T) {
	um := newTestManager(t)
	f1 := newDraftForm(t, um)
	f2 := newDraftForm(t, um)
	tg, err := um.AddTarget(context.Background(), mwanachamaforms.Target{FormID: f1.ID, ChapterID: "chapter-2"})
	if err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if err := um.RemoveTarget(context.Background(), f2.ID, tg.ID); !errors.Is(err, mwanachamaforms.ErrTargetNotFound) {
		t.Fatalf("err = %v, want ErrTargetNotFound", err)
	}
}
