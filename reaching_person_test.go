package mwanachamaforms_test

import (
	"context"
	"fmt"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"testing"
)

func groupsByPerson(held map[string][]string) forms.Groups {
	return func(_ context.Context, personID string) ([]string, error) {
		return held[personID], nil
	}
}
func publishedFormSentTo(t *testing.T, m forms.FormManager, title string, groups ...string) models.Form {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{Title: title, ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1", CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Which street?", AnswerType: models.AnswerFreeText}}); err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	for _, group := range groups {
		if _, err := m.AddTarget(ctx, models.Target{FormID: f.ID, ChapterID: group}); err != nil {
			t.Fatalf("AddTarget %s: %v", group, err)
		}
	}
	if _, _, err := m.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f
}
func titlesOf(list []models.Form) []string {
	out := []string{}
	for _, f := range list {
		out = append(out, f.Title)
	}
	return out
}
func TestAPersonIsOfferedOnlyTheFormsSentToAGroupTheyAreIn(t *testing.T) {
	ctx := context.Background()
	m := gatedManager(t, nil)
	publishedFormSentTo(t, m, "Street event permit", "g-desk")
	publishedFormSentTo(t, m, "Fence permit", "g-town")
	m.UseGroups(groupsByPerson(map[string][]string{"resident-1": {"g-town"}, "clerk-1": {"g-desk", "g-town"}}))
	resident, err := m.ListReachingPerson(ctx, "resident-1")
	if err != nil {
		t.Fatalf("ListReachingPerson: %v", err)
	}
	if got := titlesOf(resident); len(got) != 1 || got[0] != "Fence permit" {
		t.Fatalf("resident offered %v, want only the Fence permit", got)
	}
	clerk, err := m.ListReachingPerson(ctx, "clerk-1")
	if err != nil || len(clerk) != 2 {
		t.Fatalf("clerk offered %v (%v), want both forms", titlesOf(clerk), err)
	}
}
func TestAPersonReachedOnlyThroughADescendantGroupIsOffered(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	f := publishedFormSentTo(t, m, "Street event permit", "g-district")
	o, _ := s.Object("propagation")
	insert := fmt.Sprintf("insert into %s (id, form_id, chapter_id, state, targeted_at) values (?, ?, ?, 'targeted', '2026-01-01T00:00:00Z')", s.TableFor(o))
	if err := db.Exec(insert, "p-child", f.ID, "g-street").Error; err != nil {
		t.Fatalf("insert propagation: %v", err)
	}
	m.UseGroups(groupsByPerson(map[string][]string{"resident-1": {"g-street"}, "resident-2": {"g-elsewhere"}}))
	reached, err := m.ListReachingPerson(ctx, "resident-1")
	if err != nil || len(reached) != 1 {
		t.Fatalf("person in the descendant group offered %v (%v), want the form", titlesOf(reached), err)
	}
	outside, err := m.ListReachingPerson(ctx, "resident-2")
	if err != nil || len(outside) != 0 {
		t.Fatalf("person outside offered %v (%v), want none", titlesOf(outside), err)
	}
}
func TestADraftFormIsNeverOfferedAndAPersonWithNoGroupsGetsNothing(t *testing.T) {
	ctx := context.Background()
	m := gatedManager(t, nil)
	publishedFormSentTo(t, m, "Fence permit", "g-town")
	if _, err := m.Create(ctx, models.Form{Title: "Draft only", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1", CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	m.UseGroups(groupsByPerson(map[string][]string{"resident-1": {"g-town"}}))
	got, err := m.ListReachingPerson(ctx, "resident-1")
	if err != nil || len(got) != 1 || got[0].Title != "Fence permit" {
		t.Fatalf("offered %v (%v), want only the published Fence permit", titlesOf(got), err)
	}
	none, err := m.ListReachingPerson(ctx, "nobody")
	if err != nil || len(none) != 0 {
		t.Fatalf("person with no groups offered %v (%v), want none", titlesOf(none), err)
	}
}
