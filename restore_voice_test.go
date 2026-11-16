package mwanachamaforms_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func hiddenViews(t *testing.T, r voiceRound, member string) []string {
	t.Helper()
	hidden, err := r.m.ListHiddenAnswers(context.Background(), member)
	if err != nil {
		t.Fatalf("ListHiddenAnswers: %v", err)
	}
	var out []string
	for _, a := range hidden {
		out = append(out, a.ValueText)
	}
	return out
}

func TestRestoringBesideANewVoiceLeavesTheOldVoiceHidden(t *testing.T) {
	for _, voice := range []models.OneVoice{models.OneVoiceRefuse, models.OneVoiceFlag} {
		t.Run(string(voice), func(t *testing.T) {
			ctx := context.Background()
			r := openRoundForTwoGroups(t, gatedManager(t, nil), voice)
			if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
				t.Fatalf("first send: %v", err)
			}
			if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
				t.Fatalf("HideMemberAnswers: %v", err)
			}
			if err := r.sendAs("mem-1", "g-2", "close the market"); err != nil {
				t.Fatalf("send as the other group: %v", err)
			}
			if err := r.m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
				t.Fatalf("RestoreMemberAnswers beside a new voice = %v, want the old voice left hidden", err)
			}
			if got := r.sendersListed(t); len(got) != 1 {
				t.Fatalf("sends listed after the restore = %v, want the new voice alone", got)
			}
			if got := r.views(t); got["mem-1"] != "close the market" {
				t.Fatalf("answers listed after the restore = %v, want the new voice alone", got)
			}
			hidden, err := r.m.ListHiddenSends(ctx, "mem-1")
			if err != nil || len(hidden) != 1 {
				t.Fatalf("hidden sends after the restore = %+v (%v), want the old voice still hidden", hidden, err)
			}
			if got := hiddenViews(t, r, "mem-1"); len(got) != 1 || got[0] != "keep the market open" {
				t.Fatalf("hidden answers after the restore = %v, want the old voice's answer still hidden", got)
			}
		})
	}
}

func TestRestoringTwoHiddenVoicesBringsBackOnlyTheNewest(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceRefuse)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("first hide: %v", err)
	}
	if err := r.sendAs("mem-1", "g-2", "close the market"); err != nil {
		t.Fatalf("send as the other group: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("second hide: %v", err)
	}
	if err := r.m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers with two hidden voices = %v, want the newest back", err)
	}
	if got := r.sendersListed(t); len(got) != 1 {
		t.Fatalf("sends listed after the restore = %v, want one voice", got)
	}
	if got := r.views(t); got["mem-1"] != "close the market" {
		t.Fatalf("answers listed after the restore = %v, want the newest voice's answer", got)
	}
	hidden, err := r.m.ListHiddenSends(ctx, "mem-1")
	if err != nil || len(hidden) != 1 {
		t.Fatalf("hidden sends after the restore = %+v (%v), want the older voice still hidden", hidden, err)
	}
}
