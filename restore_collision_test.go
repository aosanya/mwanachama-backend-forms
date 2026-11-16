package mwanachamaforms_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestRestoringBesideANewerAnswerKeepsTheNewerOneAndLeavesTheOldHidden(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if err := r.sendAs("mem-1", "g-1", "close the market"); err != nil {
		t.Fatalf("re-answer: %v", err)
	}
	if err := r.m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers beside a newer answer = %v, want it to skip the slot", err)
	}
	if got := r.views(t); got["mem-1"] != "close the market" {
		t.Fatalf("listed after restore = %v, want the newer answer alone", got)
	}
	hidden, err := r.m.ListHiddenAnswers(ctx, "mem-1")
	if err != nil || len(hidden) != 1 || hidden[0].ValueText != "keep the market open" {
		t.Fatalf("hidden after restore = %+v (%v), want the old answer still hidden", hidden, err)
	}
}

func TestRestoringTwiceHiddenSlotBringsBackOnlyTheLatest(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("first hide: %v", err)
	}
	if err := r.sendAs("mem-1", "g-1", "close the market"); err != nil {
		t.Fatalf("re-answer: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("second hide: %v", err)
	}
	if err := r.m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers with two hidden in one slot = %v", err)
	}
	if got := r.views(t); got["mem-1"] != "close the market" {
		t.Fatalf("listed after restore = %v, want the latest hidden answer back", got)
	}
	hidden, err := r.m.ListHiddenAnswers(ctx, "mem-1")
	if err != nil || len(hidden) != 1 || hidden[0].ValueText != "keep the market open" {
		t.Fatalf("hidden after restore = %+v (%v), want the older one still hidden", hidden, err)
	}
}
