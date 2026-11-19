package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func voiceRaceRound(t *testing.T, voice models.OneVoice) (voiceRound, raceRound) {
	t.Helper()
	m, db, s := gatedManagerOn(t, nil)
	r := openRoundForTwoGroups(t, m, voice)
	return r, raceRound{m: m, db: db, s: s, form: r.form, question: r.question}
}

func TestTwoSendsAsTwoGroupsAreBothKeptWhenTheFormAllowsSeveralVoices(t *testing.T) {
	r, _ := voiceRaceRound(t, models.OneVoiceOff)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("send as g-1: %v", err)
	}
	if err := r.sendAs("mem-1", "g-2", "close the market"); err != nil {
		t.Fatalf("send as g-2: %v", err)
	}
	subs, err := r.m.ListSubmissions(context.Background(), r.form.ID, "mem-1", "", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("stored %+v, want one send for each group", subs)
	}
}

func TestASecondGroupSendThatLostTheRaceUnderRefuseIsRefusedNotStored(t *testing.T) {
	r, race := voiceRaceRound(t, models.OneVoiceRefuse)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	race.firstAttemptSeesAnEmptyWorld(t)
	if err := r.sendAs("mem-1", "g-2", "close the market"); !errors.Is(err, forms.ErrSecondVoice) {
		t.Fatalf("a send as another group that lost the race = %v, want ErrSecondVoice", err)
	}
	answers, submissions := race.stored(t, "mem-1")
	if len(answers) != 1 || answers[0].ValueText != "keep the market open" || len(submissions) != 1 {
		t.Fatalf("stored %+v and %+v, want the winner's voice alone", answers, submissions)
	}
}

func TestASecondGroupSendThatLostTheRaceUnderFlagIsKeptAsARepeat(t *testing.T) {
	r, race := voiceRaceRound(t, models.OneVoiceFlag)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	race.firstAttemptSeesAnEmptyWorld(t)
	if err := r.sendAs("mem-1", "g-2", "close the market"); err != nil {
		t.Fatalf("a send as another group that lost the race under flag = %v, want it kept as a repeat", err)
	}
	if got := r.respondents(t); got != 1 {
		t.Fatalf("respondents = %d, want 1: one voice per person", got)
	}
	_, submissions := race.stored(t, "mem-1")
	voices := 0
	for _, s := range submissions {
		if s.RepeatOf == "" {
			voices++
		}
	}
	if len(submissions) != 2 || voices != 1 {
		t.Fatalf("stored %+v, want one voice and one repeat", submissions)
	}
}
