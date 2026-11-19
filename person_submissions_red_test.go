package mwanachamaforms_test

import (
	"context"
	"encoding/json"
	"fmt"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"gorm.io/gorm"
	"testing"
)

type personSubmissionLister interface {
	ListSubmissionsOfPerson(ctx context.Context, memberID, recordRowID string) ([]models.Submission, error)
}
type personSendSeed struct {
	id, formID, memberID, submittedAt, withdrawnAt string
	answers                                        int
}

func personLister(t *testing.T, m forms.FormManager) personSubmissionLister {
	t.Helper()
	var asAny any = m
	lister, ok := asAny.(personSubmissionLister)
	if !ok {
		t.Fatal("FormManager has no ListSubmissionsOfPerson(ctx, memberID) ([]models.Submission, error)")
	}
	return lister
}
func createTitledForm(t *testing.T, m forms.FormManager, title string) models.Form {
	t.Helper()
	f, err := m.Create(context.Background(), models.Form{Title: title, ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1", CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf})
	if err != nil {
		t.Fatalf("Create %s: %v", title, err)
	}
	return f
}
func seedPersonSend(t *testing.T, db *gorm.DB, submissions, answers string, seed personSendSeed) {
	t.Helper()
	err := db.Exec("insert into "+submissions+" (id, form_id, member_id, submitted_at, withdrawn_at) values (?, ?, ?, ?, ?)", seed.id, seed.formID, seed.memberID, seed.submittedAt, seed.withdrawnAt).Error
	if err != nil {
		t.Fatalf("seed send %s: %v", seed.id, err)
	}
	for i := 0; i < seed.answers; i++ {
		err := db.Exec("insert into "+answers+" (id, form_id, question_id, member_id, value_text, answered_at, withdrawn_at) values (?, ?, ?, ?, 'yes', ?, ?)", fmt.Sprintf("%s-a%d", seed.id, i), seed.formID, fmt.Sprintf("%s-q%d", seed.id, i), seed.memberID, seed.submittedAt, seed.withdrawnAt).Error
		if err != nil {
			t.Fatalf("seed answer of %s: %v", seed.id, err)
		}
	}
}

type personRound struct {
	m               forms.FormManager
	permits, fences models.Form
}

func personSendsAcrossTwoForms(t *testing.T) personRound {
	t.Helper()
	m, db, s := gatedManagerOn(t, nil)
	permits := createTitledForm(t, m, "Building permit")
	fences := createTitledForm(t, m, "Fence height")
	submissionObject, _ := s.Object("submission")
	answerObject, _ := s.Object("answer")
	submissions, answers := s.TableFor(submissionObject), s.TableFor(answerObject)
	for _, seed := range []personSendSeed{{id: "s-old", formID: permits.ID, memberID: "mem-1", submittedAt: "2026-03-01T10:00:00Z", answers: 2}, {id: "s-new", formID: fences.ID, memberID: "mem-1", submittedAt: "2026-05-01T10:00:00Z", answers: 1}, {id: "s-gone", formID: fences.ID, memberID: "mem-1", submittedAt: "2026-06-01T10:00:00Z", withdrawnAt: "2026-06-02T10:00:00Z", answers: 1}, {id: "s-other", formID: permits.ID, memberID: "mem-2", submittedAt: "2026-04-01T10:00:00Z", answers: 3}} {
		seedPersonSend(t, db, submissions, answers, seed)
	}
	return personRound{m: m, permits: permits, fences: fences}
}
func TestAPersonsSendsAcrossFormsComeBackNewestFirstAndNothingOfAnotherMember(t *testing.T) {
	r := personSendsAcrossTwoForms(t)
	got, err := personLister(t, r.m).ListSubmissionsOfPerson(context.Background(), "mem-1", "")
	if err != nil {
		t.Fatalf("ListSubmissionsOfPerson: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("sends of mem-1 = %+v, want the two live ones", got)
	}
	if got[0].ID != "s-new" || got[1].ID != "s-old" {
		t.Fatalf("order = %s then %s, want s-new then s-old (newest first)", got[0].ID, got[1].ID)
	}
	for _, send := range got {
		if send.MemberID != "mem-1" {
			t.Errorf("send %s belongs to %s, want mem-1 only", send.ID, send.MemberID)
		}
	}
	if got[0].FormID != r.fences.ID || got[1].FormID != r.permits.ID {
		t.Errorf("forms = %s, %s, want the fence form then the permit form", got[0].FormID, got[1].FormID)
	}
	if got[0].Answers != 1 || got[1].Answers != 2 {
		t.Errorf("answer counts = %d, %d, want 1 and 2", got[0].Answers, got[1].Answers)
	}
}
func TestAPersonsSendsCarryTheTitleOfTheFormTheyWentTo(t *testing.T) {
	r := personSendsAcrossTwoForms(t)
	got, err := personLister(t, r.m).ListSubmissionsOfPerson(context.Background(), "mem-1", "")
	if err != nil {
		t.Fatalf("ListSubmissionsOfPerson: %v", err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %s, want two", raw)
	}
	if rows[0]["form_title"] != "Fence height" || rows[1]["form_title"] != "Building permit" {
		t.Fatalf("form_title of the rows = %v, %v, want Fence height then Building permit", rows[0]["form_title"], rows[1]["form_title"])
	}
}
func TestAWithdrawnSendIsLeftOutOfAPersonsSends(t *testing.T) {
	r := personSendsAcrossTwoForms(t)
	got, err := personLister(t, r.m).ListSubmissionsOfPerson(context.Background(), "mem-1", "")
	if err != nil {
		t.Fatalf("ListSubmissionsOfPerson: %v", err)
	}
	for _, send := range got {
		if send.ID == "s-gone" {
			t.Fatalf("withdrawn send s-gone listed: %+v", send)
		}
	}
}
func TestAPersonWhoSentNothingHasNoSendsAndNoError(t *testing.T) {
	r := personSendsAcrossTwoForms(t)
	got, err := personLister(t, r.m).ListSubmissionsOfPerson(context.Background(), "mem-9", "")
	if err != nil || len(got) != 0 {
		t.Fatalf("sends of mem-9 = %+v (%v), want none and no error", got, err)
	}
}
func TestAPersonsSendsNeedAMemberId(t *testing.T) {
	r := personSendsAcrossTwoForms(t)
	if _, err := personLister(t, r.m).ListSubmissionsOfPerson(context.Background(), "", ""); err == nil {
		t.Fatal("an empty member id listed sends without error")
	}
}
