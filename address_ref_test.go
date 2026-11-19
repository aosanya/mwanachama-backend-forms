package mwanachamaforms_test

import (
	"context"
	"errors"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"testing"
)

func knownPlaces(refs ...string) forms.AddressCheck {
	held := map[string]bool{}
	for _, ref := range refs {
		held[ref] = true
	}
	return func(_ context.Context, ref string) (bool, error) {
		return held[ref], nil
	}
}
func openAddressForm(t *testing.T, m forms.FormManager, allowFreeText bool) (models.Form, models.Question) {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{Title: "Synthetic street closure", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1", CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: "Which street needs to be closed?", AnswerType: models.AnswerAddress, AllowFreeText: allowFreeText}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := m.AddTarget(ctx, models.Target{FormID: f.ID, ChapterID: "g-1"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := m.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return f, q
}
func TestAPickedPlaceTheTownDoesNotHoldIsRefusedAndOneItHoldsIsKept(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseAddressCheck(knownPlaces("address:a-1", "street:Fő tér"))
	f, q := openAddressForm(t, m, false)
	ctx := context.Background()
	cases := []struct {
		name   string
		answer models.Answer
		refuse bool
	}{{"an address the town holds", models.Answer{QuestionID: q.ID, ValueText: "Fő tér 2", PlaceRef: "address:a-1"}, false}, {"a street the town holds", models.Answer{QuestionID: q.ID, ValueText: "Fő tér", PlaceRef: "street:Fő tér"}, false}, {"an address the town does not hold", models.Answer{QuestionID: q.ID, ValueText: "Fő tér 2", PlaceRef: "address:nowhere"}, true}, {"a street the town does not hold", models.Answer{QuestionID: q.ID, ValueText: "Fő tér", PlaceRef: "street:x"}, true}}
	for i, c := range cases {
		member := "mem-" + string(rune('1'+i))
		_, err := m.SubmitAnswersAs(ctx, f.ID, member, "g-1", []models.Answer{c.answer})
		if c.refuse && !errors.Is(err, forms.ErrInvalidAnswer) {
			t.Errorf("%s: want ErrInvalidAnswer, got %v", c.name, err)
		}
		if !c.refuse && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
	}
}
func TestFreeTextIsKeptWhereAllowedWithoutAnyLookup(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseAddressCheck(knownPlaces())
	f, q := openAddressForm(t, m, true)
	_, err := m.SubmitAnswersAs(context.Background(), f.ID, "mem-1", "g-1", []models.Answer{{QuestionID: q.ID, ValueText: "the car park behind the mill"}})
	if err != nil {
		t.Fatalf("free text where allowed: %v", err)
	}
}
func TestACorrectionMeetsTheSameCheck(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseAddressCheck(knownPlaces("address:a-1"))
	f, q := openAddressForm(t, m, false)
	ctx := context.Background()
	if _, err := m.SubmitAnswersAs(ctx, f.ID, "mem-1", "g-1", []models.Answer{{QuestionID: q.ID, ValueText: "Fő tér 2", PlaceRef: "address:a-1"}}); err != nil {
		t.Fatalf("send: %v", err)
	}
	_, err := m.CorrectSubmission(ctx, f.ID, "mem-1", "clerk-1", []models.Answer{{QuestionID: q.ID, ValueText: "Fő tér 9", PlaceRef: "address:nowhere"}})
	if !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("a correction naming an unknown place: want ErrInvalidAnswer, got %v", err)
	}
}
func TestAManagerWithoutACheckKeepsTheShapeRuleOnly(t *testing.T) {
	m := gatedManager(t, nil)
	f, q := openAddressForm(t, m, false)
	_, err := m.SubmitAnswersAs(context.Background(), f.ID, "mem-1", "g-1", []models.Answer{{QuestionID: q.ID, ValueText: "Fő tér 2", PlaceRef: "address:anything"}})
	if err != nil {
		t.Fatalf("no check set: %v", err)
	}
}
