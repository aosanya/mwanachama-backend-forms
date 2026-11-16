package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func publicRound(t *testing.T, m forms.FormManager) (models.Form, string) {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic open survey", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudiencePublic, CollectionMode: models.CollectionSelf,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Ward", AnswerType: models.AnswerFreeText}}); err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	published, link, err := m.Publish(ctx, f.ID, "", "author-1", "")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return published, link.Key
}

func TestAMemberClaimsTheirAnonymousRespondentOnce(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	_, key := publicRound(t, m)

	seen, err := m.RegisterRespondent(ctx, key, "")
	if err != nil || seen.PublicKey == "" {
		t.Fatalf("RegisterRespondent = %+v, %v", seen, err)
	}

	claimed, err := m.ClaimRespondent(ctx, key, seen.PublicKey, "mem-1")
	if err != nil {
		t.Fatalf("ClaimRespondent: %v", err)
	}
	if claimed.PublicKey != seen.PublicKey {
		t.Fatalf("the claim answered for another respondent: %+v", claimed)
	}

	again, err := m.ClaimRespondent(ctx, key, seen.PublicKey, "mem-1")
	if err != nil || again.PublicKey != seen.PublicKey {
		t.Fatalf("the same member claiming again must succeed quietly: %+v, %v", again, err)
	}

	if _, err := m.ClaimRespondent(ctx, key, seen.PublicKey, "mem-2"); !errors.Is(err, forms.ErrRespondentClaimed) {
		t.Fatalf("another member claiming = %v, want ErrRespondentClaimed", err)
	}
}

func TestAClaimNeedsASignedInMemberAKnownLinkAndAKnownRespondent(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	_, key := publicRound(t, m)
	seen, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}

	if _, err := m.ClaimRespondent(ctx, key, seen.PublicKey, ""); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("a claim by nobody = %v, want ErrInvalidReference", err)
	}
	if _, err := m.ClaimRespondent(ctx, "no-such-link", seen.PublicKey, "mem-1"); !errors.Is(err, forms.ErrLinkNotFound) {
		t.Fatalf("a claim through an unknown link = %v, want ErrLinkNotFound", err)
	}
	if _, err := m.ClaimRespondent(ctx, key, "", "mem-1"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("a claim without a public key = %v, want ErrInvalidReference", err)
	}
	if _, err := m.ClaimRespondent(ctx, key, "nobody-holds-this", "mem-1"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("a claim of a key nobody holds = %v, want ErrInvalidReference", err)
	}
}

func TestRegisteringTheSameKeyTwiceGivesTheSameRespondent(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	_, key := publicRound(t, m)
	first, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	second, err := m.RegisterRespondent(ctx, key, first.PublicKey)
	if err != nil || second.PublicKey != first.PublicKey {
		t.Fatalf("a known key must give the same respondent: %+v, %v", second, err)
	}
	if _, err := m.RegisterRespondent(ctx, "no-such-link", ""); !errors.Is(err, forms.ErrLinkNotFound) {
		t.Fatalf("registering through an unknown link = %v, want ErrLinkNotFound", err)
	}
}

func TestADeclarationIsOnePerRespondentSoDeclaringAgainMovesIt(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	f, key := publicRound(t, m)
	m.UseDeclarable(func(_ context.Context, formID string) ([]string, error) {
		if formID != f.ID {
			t.Errorf("the offer was asked about form %s, not %s", formID, f.ID)
		}
		return []string{"g-east", "g-west"}, nil
	})

	one, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	two, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	respondentID := func(view models.RespondentPublicView) string { return view.ID }
	for _, d := range []struct{ who, group string }{
		{respondentID(one), "g-east"}, {respondentID(two), "g-east"}, {respondentID(two), "g-west"},
	} {
		if _, err := m.DeclareChapter(ctx, key, d.who, d.group, ""); err != nil {
			t.Fatalf("DeclareChapter %s: %v", d.group, err)
		}
	}

	if _, err := m.DeclareChapter(ctx, key, respondentID(one), "g-north", ""); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("a group the form does not offer = %v, want ErrInvalidReference", err)
	}

	counts, err := m.CountDeclarations(ctx, f.ID)
	if err != nil {
		t.Fatalf("CountDeclarations: %v", err)
	}
	if len(counts) != 2 || counts[0].ChapterID != "g-east" || counts[0].Count != 1 || counts[1].ChapterID != "g-west" || counts[1].Count != 1 {
		t.Fatalf("counts = %+v, want g-east 1 then g-west 1 (the second respondent moved from east to west)", counts)
	}

	listed, err := m.ListDeclarations(ctx, f.ID)
	if err != nil || len(listed) != 2 {
		t.Fatalf("ListDeclarations = %d rows, %v, want one for each respondent", len(listed), err)
	}
	none, err := m.ListDeclarations(ctx, "other-form")
	if err != nil || len(none) != 0 {
		t.Fatalf("another form has no declarations: %d rows, %v", len(none), err)
	}
}
