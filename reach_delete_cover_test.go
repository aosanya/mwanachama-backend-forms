package mwanachamaforms_test

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func reachForm(t *testing.T, m forms.FormManager, origin string, groups ...string) models.Form {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic round " + origin, ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: origin,
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Choose", AnswerType: models.AnswerSingleChoice}, Options: []models.QuestionOption{{Label: "yes"}, {Label: "no"}}}); err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	return f
}

func target(t *testing.T, m forms.FormManager, f models.Form, group string, down bool) {
	t.Helper()
	if _, err := m.AddTarget(context.Background(), models.Target{FormID: f.ID, ChapterID: group, IncludesDescendants: down}); err != nil {
		t.Fatalf("AddTarget %s: %v", group, err)
	}
}

func publishForm(t *testing.T, m forms.FormManager, f models.Form) {
	t.Helper()
	if _, _, err := m.Publish(context.Background(), f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func idsOf(list []models.Form) []string {
	out := []string{}
	for _, f := range list {
		out = append(out, f.ID)
	}
	sort.Strings(out)
	return out
}

func TestAGroupSeesTheFormsItStartedAndNoOthers(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	a := reachForm(t, m, "g-1")
	b := reachForm(t, m, "g-1")
	reachForm(t, m, "g-2")

	got, err := m.ListForChapter(ctx, "g-1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{a.ID, b.ID}
	sort.Strings(want)
	if !reflect.DeepEqual(idsOf(got), want) {
		t.Errorf("g-1 started %v, got %v", want, idsOf(got))
	}
	if len(got) == 2 && got[0].CreatedAt > got[1].CreatedAt {
		t.Errorf("the oldest comes first: %s then %s", got[0].CreatedAt, got[1].CreatedAt)
	}
	none, err := m.ListForChapter(ctx, "g-404")
	if err != nil || len(none) != 0 {
		t.Errorf("a group that started none: %v, %v", none, err)
	}
}

func TestAMemberReachesOpenFormsAimedAtTheirGroupAndAtParentsThatReachDown(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()

	own := reachForm(t, m, "g-2")
	target(t, m, own, "g-2", false)
	publishForm(t, m, own)

	fromAbove := reachForm(t, m, "g-1")
	target(t, m, fromAbove, "g-1", true)
	publishForm(t, m, fromAbove)

	notDown := reachForm(t, m, "g-1")
	target(t, m, notDown, "g-1", false)
	publishForm(t, m, notDown)

	both := reachForm(t, m, "g-1")
	target(t, m, both, "g-2", false)
	target(t, m, both, "g-1", true)
	publishForm(t, m, both)

	draft := reachForm(t, m, "g-2")
	target(t, m, draft, "g-2", false)

	elsewhere := reachForm(t, m, "g-9")
	target(t, m, elsewhere, "g-9", true)
	publishForm(t, m, elsewhere)

	got, err := m.ListReachable(ctx, []string{"g-2", "g-1"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{own.ID, fromAbove.ID, both.ID}
	sort.Strings(want)
	if !reflect.DeepEqual(idsOf(got), want) {
		t.Errorf("reachable from g-2 under g-1: want %v, got %v", want, idsOf(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ID > got[i].ID {
			t.Errorf("sorted by id: %v", idsOf(got))
		}
	}

	nobody, err := m.ListReachable(ctx, nil)
	if err != nil || nobody == nil || len(nobody) != 0 {
		t.Errorf("no ancestry reaches nothing, as an empty list: %v, %v", nobody, err)
	}
}

func TestAClosedFormStaysReachableButADeletedOneDoesNot(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	f := reachForm(t, m, "g-2")
	target(t, m, f, "g-2", false)
	publishForm(t, m, f)
	if _, err := m.Close(ctx, f.ID, "author-1"); err != nil {
		t.Fatal(err)
	}
	got, err := m.ListReachable(ctx, []string{"g-2"})
	if err != nil || !reflect.DeepEqual(idsOf(got), []string{f.ID}) {
		t.Errorf("a closed form is still visible to members: %v, %v", idsOf(got), err)
	}
}

func TestOnlyAFormThatWasNeverPublishedCanBeDeleted(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()

	draft := reachForm(t, m, "g-1")
	target(t, m, draft, "g-1", false)
	if err := m.Delete(ctx, draft.ID); err != nil {
		t.Fatalf("Delete a draft: %v", err)
	}
	if _, err := m.Get(ctx, draft.ID); !errors.Is(err, forms.ErrFormNotFound) {
		t.Errorf("the deleted draft is gone: %v", err)
	}
	if qs, err := m.ListQuestions(ctx, draft.ID); err != nil || len(qs) != 0 {
		t.Errorf("its questions went with it: %v, %v", qs, err)
	}
	if ts, err := m.ListTargets(ctx, draft.ID); err != nil || len(ts) != 0 {
		t.Errorf("its targets went with it: %v, %v", ts, err)
	}

	open := reachForm(t, m, "g-1")
	target(t, m, open, "g-1", false)
	publishForm(t, m, open)
	if err := m.Delete(ctx, open.ID); !errors.Is(err, forms.ErrPublished) {
		t.Errorf("deleting an open form: %v", err)
	}
	if _, err := m.Close(ctx, open.ID, "author-1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(ctx, open.ID); !errors.Is(err, forms.ErrPublished) {
		t.Errorf("deleting a closed form: %v", err)
	}
	if _, err := m.Get(ctx, open.ID); err != nil {
		t.Errorf("a refused delete leaves the form: %v", err)
	}

	if err := m.Delete(ctx, "no-such-form"); !errors.Is(err, forms.ErrFormNotFound) {
		t.Errorf("deleting a stranger: %v", err)
	}
}

func TestAPublicLinkKeyCanBeUsedOnlyOnce(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := context.Background()
	publicForm := func() models.Form {
		f, err := m.Create(ctx, models.Form{
			Title: "Synthetic open survey", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
			CreatedBy: "author-1", Audience: models.AudiencePublic, CollectionMode: models.CollectionSelf,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Ward", AnswerType: models.AnswerFreeText}}); err != nil {
			t.Fatal(err)
		}
		return f
	}

	first := publicForm()
	_, link, err := m.Publish(ctx, first.ID, "ward-survey", "author-1", "")
	if err != nil || link.Key != "ward-survey" || link.FormID != first.ID || link.Status != models.LinkActive || link.CreatedBy != "author-1" {
		t.Fatalf("first publish: %+v, %v", link, err)
	}

	second := publicForm()
	if _, _, err := m.Publish(ctx, second.ID, "ward-survey", "author-1", ""); !errors.Is(err, forms.ErrLinkKeyTaken) {
		t.Fatalf("the same key on another form: %v", err)
	}
	after, err := m.Get(ctx, second.ID)
	if err != nil || after.Status != models.StatusDraft {
		t.Errorf("a refused key leaves the form a draft: %+v, %v", after, err)
	}
}
