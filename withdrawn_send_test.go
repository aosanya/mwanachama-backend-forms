package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (r voiceRound) sendersListed(t *testing.T) []string {
	t.Helper()
	listed, err := r.m.ListSubmissions(context.Background(), r.form.ID, "", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	var out []string
	for _, s := range listed {
		out = append(out, s.MemberID)
	}
	return out
}

func TestWithdrawalHidesTheSendAndRestoreBringsItBack(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := r.sendAs("mem-2", "g-1", "close the market"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if got := r.sendersListed(t); len(got) != 1 || got[0] != "mem-2" {
		t.Fatalf("sends listed after the withdrawal = %v, want mem-2 alone", got)
	}
	if got := r.respondents(t); got != 1 {
		t.Fatalf("respondents after the withdrawal = %d, want 1", got)
	}
	hidden, err := r.m.ListHiddenSends(ctx, "mem-1")
	if err != nil || len(hidden) != 1 || hidden[0].MemberID != "mem-1" || hidden[0].WithdrawnAt == "" || hidden[0].SubmittedAt == "" {
		t.Fatalf("hidden sends = %+v (%v), want mem-1's send with its time and a stamp", hidden, err)
	}
	if other, err := r.m.ListHiddenSends(ctx, "mem-2"); err != nil || len(other) != 0 {
		t.Fatalf("hidden sends of a member who did not withdraw = %+v (%v), want none", other, err)
	}
	if _, err := r.m.ListHiddenSends(ctx, ""); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("ListHiddenSends empty = %v, want ErrInvalidAnswer", err)
	}
	if _, err := r.m.MarkPaid(ctx, r.form.ID, "mem-1", true, "clerk-1"); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("MarkPaid on a withdrawn send = %v, want ErrSubmissionNotFound", err)
	}
	if err := r.m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers: %v", err)
	}
	if got := r.sendersListed(t); len(got) != 2 {
		t.Fatalf("sends listed after the restore = %v, want both back", got)
	}
	if got := r.respondents(t); got != 2 {
		t.Fatalf("respondents after the restore = %d, want 2", got)
	}
}

func TestASendAfterTheWithdrawalSitsBesideTheHiddenOneAndRestoreLeavesItAlone(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if err := r.sendAs("mem-1", "g-1", "close the market"); err != nil {
		t.Fatalf("send after the withdrawal: %v", err)
	}
	if got := r.sendersListed(t); len(got) != 1 {
		t.Fatalf("sends listed after the new send = %v, want the new one alone", got)
	}
	if err := r.m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers beside a newer send = %v", err)
	}
	if got := r.sendersListed(t); len(got) != 1 {
		t.Fatalf("sends listed after the restore = %v, want the newer one alone", got)
	}
	hidden, err := r.m.ListHiddenSends(ctx, "mem-1")
	if err != nil || len(hidden) != 1 {
		t.Fatalf("hidden sends after the restore = %+v (%v), want the old send still hidden", hidden, err)
	}
}

func TestAWithdrawnSendDoesNotMakeASecondVoiceAfterwards(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceRefuse)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if err := r.sendAs("mem-1", "g-2", "close the market"); err != nil {
		t.Fatalf("send after the withdrawal under one voice = %v, want it accepted as the voice", err)
	}
}
