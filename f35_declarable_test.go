package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestF35ADeclarationOutsideTheOfferedGroupsIsRefusedAndTheRestAreCounted(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	f := newOpenPublicForm(t, um, "ward-key")
	um.UseDeclarable(func(_ context.Context, formID string) ([]string, error) {
		if formID != f.ID {
			t.Fatalf("asked for the groups of %q, want %q", formID, f.ID)
		}
		return []string{"ward-1", "ward-2"}, nil
	})

	respond := func(pk string) string {
		r, err := um.RegisterRespondent(ctx, "ward-key", pk)
		if err != nil {
			t.Fatalf("RegisterRespondent: %v", err)
		}
		return r.ID
	}
	a, b, c, d := respond("pk-a"), respond("pk-b"), respond("pk-c"), respond("pk-d")

	if _, err := um.DeclareChapter(ctx, "ward-key", a, "ward-9", ""); !errors.Is(err, mwanachamaforms.ErrInvalidReference) {
		t.Fatalf("a group the form does not offer: err = %v, want ErrInvalidReference", err)
	}
	for _, step := range []struct{ who, group, text string }{{a, "ward-1", ""}, {b, "ward-1", ""}, {c, "ward-2", ""}, {d, "", "my street"}} {
		if _, err := um.DeclareChapter(ctx, "ward-key", step.who, step.group, step.text); err != nil {
			t.Fatalf("DeclareChapter(%q): %v", step.group, err)
		}
	}
	if _, err := um.DeclareChapter(ctx, "no-such-key", a, "ward-9", ""); !errors.Is(err, mwanachamaforms.ErrLinkNotFound) {
		t.Fatalf("an unknown key still answers as not found before any group is checked: %v", err)
	}

	listed, err := um.ListDeclarations(ctx, f.ID)
	if err != nil || len(listed) != 4 {
		t.Fatalf("ListDeclarations = %d %v, want 4", len(listed), err)
	}
	counts, err := um.CountDeclarations(ctx, f.ID)
	if err != nil {
		t.Fatalf("CountDeclarations: %v", err)
	}
	want := map[string]int{"": 1, "ward-1": 2, "ward-2": 1}
	if len(counts) != len(want) {
		t.Fatalf("counts = %+v, want %v", counts, want)
	}
	for _, got := range counts {
		if want[got.ChapterID] != got.Count {
			t.Fatalf("counts = %+v, want %v", counts, want)
		}
	}
}

func TestF35WithNoDeclarableResolverAnyGroupIsRecorded(t *testing.T) {
	ctx := context.Background()
	um := newTestManager(t)
	newOpenPublicForm(t, um, "open-key")
	r, err := um.RegisterRespondent(ctx, "open-key", "pk")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	if _, err := um.DeclareChapter(ctx, "open-key", r.ID, "ward-9", ""); err != nil {
		t.Fatalf("DeclareChapter with no resolver: %v", err)
	}
}
