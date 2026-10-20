package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

type linkedSurvey struct {
	form     mwanachamaforms.Form
	text     mwanachamaforms.Question
	choice   mwanachamaforms.Question
	yesNo    mwanachamaforms.Question
	firstKey string
}

func composeLinkedSurvey(t *testing.T, mgr mwanachamaforms.FormManager, key string) linkedSurvey {
	t.Helper()
	ctx := context.Background()
	f, err := mgr.Create(ctx, mwanachamaforms.Form{
		Title: "Street lighting", OriginatorChapterID: "county-1", ClosesAt: futureRFC3339(time.Hour), Audience: mwanachamaforms.AudiencePublic,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	add := func(draft mwanachamaforms.QuestionDraft) mwanachamaforms.Question {
		draft.FormID = f.ID
		q, _, err := mgr.AddQuestion(ctx, draft)
		if err != nil {
			t.Fatalf("AddQuestion: %v", err)
		}
		return q
	}
	out := linkedSurvey{firstKey: key}
	out.text = add(mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "What is dark?"}})
	out.choice = add(mwanachamaforms.QuestionDraft{
		Question: mwanachamaforms.Question{AnswerType: mwanachamaforms.AnswerSingleChoice, Prompt: "Which street?"},
		Options:  []mwanachamaforms.QuestionOption{{Label: "Main"}, {Label: "Market"}},
	})
	out.yesNo = add(mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{AnswerType: mwanachamaforms.AnswerYesNo, Prompt: "Is it urgent?"}})
	published, _, err := mgr.Publish(ctx, f.ID, key, "sender-1", "")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	out.form = published
	return out
}

func TestALinksQuestionsComeInOrderWithTheirOptions(t *testing.T) {
	mgr := newTestManager(t)
	s := composeLinkedSurvey(t, mgr, "link-a")

	got, err := mgr.PublicLinkQuestions(context.Background(), "link-a")
	if err != nil {
		t.Fatalf("PublicLinkQuestions: %v", err)
	}
	want := []string{s.text.ID, s.choice.ID, s.yesNo.ID}
	if len(got) != len(want) {
		t.Fatalf("got %d questions, want %d: %+v", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("question %d is %q, want %q", i, got[i].ID, id)
		}
	}
	if len(got[0].Options) != 0 || len(got[2].Options) != 0 {
		t.Fatalf("a question without choices came with options: %+v", got)
	}
	opts := got[1].Options
	if len(opts) != 2 || opts[0].Label != "Main" || opts[1].Label != "Market" {
		t.Fatalf("choice options = %+v, want Main then Market", opts)
	}
}

func TestALinksQuestionsLeaveOutSupersededVersions(t *testing.T) {
	mgr := newTestManager(t)
	s := composeLinkedSurvey(t, mgr, "link-a")

	next, err := mgr.VersionQuestion(context.Background(), s.choice.ID, "sender-1", mwanachamaforms.QuestionDraft{
		Question: mwanachamaforms.Question{Prompt: "Which road?"},
		Options:  []mwanachamaforms.QuestionOption{{Label: "Ring road"}, {Label: "Bypass"}, {Label: "Main"}},
	})
	if err != nil {
		t.Fatalf("VersionQuestion: %v", err)
	}

	got, err := mgr.PublicLinkQuestions(context.Background(), "link-a")
	if err != nil {
		t.Fatalf("PublicLinkQuestions: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d questions, want 3: %+v", len(got), got)
	}
	for _, q := range got {
		if q.ID == s.choice.ID {
			t.Fatalf("the superseded version %q is still offered", s.choice.ID)
		}
	}
	if got[1].ID != next.Question.ID || got[1].Prompt != "Which road?" {
		t.Fatalf("second question = %+v, want the current version %q", got[1].Question, next.Question.ID)
	}
	if len(got[1].Options) != 3 || got[1].Options[0].Label != "Ring road" {
		t.Fatalf("current version's options = %+v", got[1].Options)
	}
}

func TestALinksQuestionsAreThatFormsAlone(t *testing.T) {
	mgr := newTestManager(t)
	a := composeLinkedSurvey(t, mgr, "link-a")
	b := composeLinkedSurvey(t, mgr, "link-b")

	got, err := mgr.PublicLinkQuestions(context.Background(), "link-a")
	if err != nil {
		t.Fatalf("PublicLinkQuestions: %v", err)
	}
	for _, q := range got {
		if q.FormID != a.form.ID {
			t.Fatalf("link-a offered question %q from form %q", q.ID, q.FormID)
		}
		for _, other := range []string{b.text.ID, b.choice.ID, b.yesNo.ID} {
			if q.ID == other {
				t.Fatalf("link-a offered link-b's question %q", other)
			}
		}
	}
}

func TestALinksQuestionsRefuseEveryMissInTheSameWords(t *testing.T) {
	cases := map[string]func(t *testing.T, mgr mwanachamaforms.FormManager, db *gorm.DB, s linkedSurvey){
		"unknown key": func(t *testing.T, mgr mwanachamaforms.FormManager, db *gorm.DB, s linkedSurvey) {},
		"retired link": func(t *testing.T, mgr mwanachamaforms.FormManager, db *gorm.DB, s linkedSurvey) {
			setColumn(t, db, "public_link", "key", s.firstKey, "status", "retired")
		},
		"closed form": func(t *testing.T, mgr mwanachamaforms.FormManager, db *gorm.DB, s linkedSurvey) {
			if _, err := mgr.Close(context.Background(), s.form.ID, "sender-1"); err != nil {
				t.Fatalf("Close: %v", err)
			}
		},
		"form not public": func(t *testing.T, mgr mwanachamaforms.FormManager, db *gorm.DB, s linkedSurvey) {
			setColumn(t, db, "form", "id", s.form.ID, "audience", "member")
		},
		"missing form": func(t *testing.T, mgr mwanachamaforms.FormManager, db *gorm.DB, s linkedSurvey) {
			setColumn(t, db, "public_link", "key", s.firstKey, "form_id", "no-such-form")
		},
	}
	for name, spoil := range cases {
		t.Run(name, func(t *testing.T) {
			mgr, db := newTestManagerDB(t)
			s := composeLinkedSurvey(t, mgr, "link-a")
			spoil(t, mgr, db, s)

			key := s.firstKey
			if name == "unknown key" {
				key = "link-z"
			}
			got, err := mgr.PublicLinkQuestions(context.Background(), key)
			if !errors.Is(err, mwanachamaforms.ErrLinkNotFound) || err.Error() != mwanachamaforms.ErrLinkNotFound.Error() {
				t.Fatalf("err = %v, want exactly ErrLinkNotFound", err)
			}
			if got != nil {
				t.Fatalf("a refused link still offered %+v", got)
			}
		})
	}
}

func setColumn(t *testing.T, db *gorm.DB, role, keyColumn, keyValue, column, value string) {
	t.Helper()
	s := testSpec(t, "test")
	o, _ := s.ByRole(role)
	res := db.Table(s.TableFor(o)).Where(keyColumn+" = ?", keyValue).Update(column, value)
	if res.Error != nil || res.RowsAffected != 1 {
		t.Fatalf("set %s.%s: %v (rows %d)", role, column, res.Error, res.RowsAffected)
	}
}

func TestALinksQuestionsAreReadWhateverGroupsTheCallerIsIn(t *testing.T) {
	mgr := newTestManager(t)
	s := composeLinkedSurvey(t, mgr, "link-a")

	ctx := mwanachamaforms.OnlyReached(context.Background(), "stranger")
	got, err := mgr.PublicLinkQuestions(ctx, "link-a")
	if err != nil {
		t.Fatalf("PublicLinkQuestions narrowed to a stranger: %v", err)
	}
	if len(got) != 3 || got[0].ID != s.text.ID || len(got[1].Options) != 2 {
		t.Fatalf("narrowed read = %+v, want the link's three questions with their options", got)
	}
}
