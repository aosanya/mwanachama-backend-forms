package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type voiceRound struct {
	m        forms.FormManager
	form     models.Form
	question models.Question
	reason   models.Question
}

func openRoundForTwoGroups(t *testing.T, m forms.FormManager, voice models.OneVoice) voiceRound {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic consultation", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
		OneVoice: voice,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Your view", AnswerType: models.AnswerFreeText}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	reason, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Why", AnswerType: models.AnswerFreeText}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	for _, group := range []string{"g-1", "g-2"} {
		if _, err := m.AddTarget(ctx, models.Target{FormID: f.ID, ChapterID: group}); err != nil {
			t.Fatalf("AddTarget %s: %v", group, err)
		}
	}
	if _, _, err := m.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return voiceRound{m: m, form: f, question: q, reason: reason}
}

func (r voiceRound) sendAs(member, group, view string) error {
	_, err := r.m.SubmitAnswersAs(context.Background(), r.form.ID, member, group, []models.Answer{{QuestionID: r.question.ID, ValueText: view}})
	return err
}

func (r voiceRound) views(t *testing.T) map[string]string {
	t.Helper()
	listed, err := r.m.ListAllAnswers(context.Background(), r.form.ID, "", "")
	if err != nil {
		t.Fatalf("ListAllAnswers: %v", err)
	}
	out := map[string]string{}
	for _, a := range listed {
		out[a.MemberID] = a.ValueText
	}
	return out
}

func (r voiceRound) respondents(t *testing.T) int {
	t.Helper()
	page, err := r.m.Register(context.Background(), models.RegisterQuery{})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return page.Respondents
}

func TestARefusedSecondVoiceLeavesTheFirstAnswerStanding(t *testing.T) {
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceRefuse)
	if err := r.sendAs("mem-1", "g-1", "keep the market open"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.sendAs("mem-1", "g-1", "close the market"); !errors.Is(err, forms.ErrSecondVoice) {
		t.Fatalf("resend as the same group = %v, want ErrSecondVoice", err)
	}
	if err := r.sendAs("mem-1", "g-2", "close the market"); !errors.Is(err, forms.ErrSecondVoice) {
		t.Fatalf("send as another group = %v, want ErrSecondVoice", err)
	}
	if err := r.sendAs("mem-2", "g-1", "close the market"); err != nil {
		t.Fatalf("another person's send = %v, want accepted", err)
	}
	if got := r.views(t); len(got) != 2 || got["mem-1"] != "keep the market open" || got["mem-2"] != "close the market" {
		t.Fatalf("answers = %v, want each person's first", got)
	}
	if got := r.respondents(t); got != 2 {
		t.Fatalf("respondents = %d, want 2", got)
	}
}

func TestAFlaggedSecondVoiceIsKeptBesideTheFirstAndNeverCounted(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceFlag)
	steps := []struct{ group, view string }{{"g-1", "keep the market open"}, {"g-2", "close the market"}, {"g-2", "close it on Sundays"}, {"g-1", "keep it open"}}
	for _, step := range steps {
		if err := r.sendAs("mem-1", step.group, step.view); err != nil {
			t.Fatalf("send as %s: %v", step.group, err)
		}
	}
	if got := r.views(t); len(got) != 1 || got["mem-1"] != "keep the market open" {
		t.Fatalf("listed answers = %v, want only the first voice", got)
	}
	if got := r.respondents(t); got != 1 {
		t.Fatalf("respondents = %d, want 1", got)
	}
	sent, err := r.m.ListSubmissions(ctx, r.form.ID, "mem-1", "")
	if err != nil {
		t.Fatalf("ListSubmissions: %v", err)
	}
	if len(sent) != 3 {
		t.Fatalf("stored sends = %d, want the voice and two repeats", len(sent))
	}
	voice := ""
	for _, s := range sent {
		if s.RepeatOf == "" {
			voice = s.ID
		}
	}
	for _, s := range sent {
		if s.RepeatOf != "" && s.RepeatOf != voice {
			t.Fatalf("repeat %+v does not point at the voice %s", s, voice)
		}
		if s.Answers != 1 {
			t.Fatalf("send %+v counts %d answers, want its own 1", s, s.Answers)
		}
	}
	for _, s := range sent {
		got, err := r.m.GetSend(ctx, r.form.ID, s.ID, "mem-1")
		if err != nil || len(got.Answers) != 1 {
			t.Fatalf("GetSend %s = %+v (%v), want its own answer", s.ID, got, err)
		}
		if s.RepeatOf == "" && got.Answers[0].ValueText != "keep the market open" {
			t.Fatalf("the voice reads %q", got.Answers[0].ValueText)
		}
		if s.RepeatOf != "" && got.Answers[0].RepeatOf != voice {
			t.Fatalf("a repeat reads %+v", got.Answers[0])
		}
	}
}

func TestAFormWithoutOneVoiceStillLetsAResendEdit(t *testing.T) {
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	for _, view := range []string{"keep the market open", "close the market"} {
		if err := r.sendAs("mem-1", "g-1", view); err != nil {
			t.Fatalf("send: %v", err)
		}
	}
	if got := r.views(t); got["mem-1"] != "close the market" {
		t.Fatalf("answers = %v, want the edit", got)
	}
}

func TestOneVoiceIsRefusedWhereThereIsNoVerifiedContact(t *testing.T) {
	m := gatedManager(t, nil)
	base := models.Form{Title: "Synthetic", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1", CreatedBy: "author-1"}
	cases := map[string]func(f models.Form) models.Form{
		"public audience": func(f models.Form) models.Form {
			f.Audience = models.AudiencePublic
			f.OneVoice = models.OneVoiceRefuse
			return f
		},
		"standing":       func(f models.Form) models.Form { f.Standing = true; f.OneVoice = models.OneVoiceFlag; return f },
		"unknown policy": func(f models.Form) models.Form { f.OneVoice = "ignore"; return f },
	}
	for name, make := range cases {
		if _, err := m.Create(context.Background(), make(base)); !errors.Is(err, forms.ErrInvalidForm) {
			t.Fatalf("%s: Create = %v, want ErrInvalidForm", name, err)
		}
	}
}

func TestAnAnswerHiddenByAWithdrawalIsNeverOverwrittenByAReAnswer(t *testing.T) {
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
	if got := r.views(t); got["mem-1"] != "close the market" {
		t.Fatalf("listed = %v, want the new answer", got)
	}
	hidden, err := r.m.ListHiddenAnswers(ctx, "mem-1")
	if err != nil || len(hidden) != 1 || hidden[0].ValueText != "keep the market open" || hidden[0].WithdrawnAt == "" {
		t.Fatalf("hidden = %+v (%v), want the old answer still hidden and stored", hidden, err)
	}
	if err := r.sendAs("mem-1", "g-1", "close it on Sundays"); err != nil {
		t.Fatalf("edit of the new answer: %v", err)
	}
	if hidden, err = r.m.ListHiddenAnswers(ctx, "mem-1"); err != nil || len(hidden) != 1 || hidden[0].ValueText != "keep the market open" {
		t.Fatalf("hidden after an edit = %+v (%v), want the old answer untouched", hidden, err)
	}
}

func TestAHiddenAnswerToAPromptTheReAnswerLeavesOutIsNotDeleted(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	both := []models.Answer{{QuestionID: r.question.ID, ValueText: "keep the market open"}, {QuestionID: r.reason.ID, ValueText: "it feeds the street"}}
	if _, err := r.m.SubmitAnswersAs(ctx, r.form.ID, "mem-1", "g-1", both); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if err := r.sendAs("mem-1", "g-1", "close the market"); err != nil {
		t.Fatalf("re-answer: %v", err)
	}
	hidden, err := r.m.ListHiddenAnswers(ctx, "mem-1")
	if err != nil || len(hidden) != 2 {
		t.Fatalf("hidden = %+v (%v), want both old answers stored", hidden, err)
	}
}
