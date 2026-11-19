package mwanachamaforms_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

func TestReleasingAMemberUnlinksTheirLinkAnswersOnly(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	o, ok := s.Object("respondent")
	if !ok {
		t.Fatal("no respondent object")
	}
	table := s.TableFor(o)
	insert := fmt.Sprintf("insert into %s (id, public_key, first_seen_at, claimed_by_member_id, claimed_at) values (?, ?, '2026-01-01T00:00:00Z', ?, '2026-01-02T00:00:00Z')", table)
	for _, row := range [][]string{{"r-1", "k-1", "mem-1"}, {"r-2", "k-2", "mem-2"}} {
		if err := db.Exec(insert, row[0], row[1], row[2]).Error; err != nil {
			t.Fatalf("seed respondent: %v", err)
		}
	}
	if err := m.ReleaseMember(ctx, "mem-1"); err != nil {
		t.Fatalf("ReleaseMember: %v", err)
	}
	claimed := func(id string) string {
		var who *string
		if err := db.Raw("select claimed_by_member_id from "+table+" where id = ?", id).Scan(&who).Error; err != nil {
			t.Fatalf("read respondent: %v", err)
		}
		if who == nil {
			return ""
		}
		return *who
	}
	if got := claimed("r-1"); got != "" {
		t.Fatalf("released respondent still claimed by %q", got)
	}
	if got := claimed("r-2"); got != "mem-2" {
		t.Fatalf("other member's respondent claimed by %q, want mem-2", got)
	}
}

func TestReleasingNobodyIsRefused(t *testing.T) {
	m := gatedManager(t, nil)
	if err := m.ReleaseMember(context.Background(), ""); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("ReleaseMember empty = %v, want ErrInvalidAnswer", err)
	}
}

func TestWithdrawalHidesTheMembersAnswersWithoutDeletingThem(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	o, _ := s.Object("answer")
	table := s.TableFor(o)
	insert := fmt.Sprintf("insert into %s (id, form_id, question_id, member_id, value_text, answered_at) values (?, 'f-1', 'q-1', ?, 'yes', '2026-01-01T00:00:00Z')", table)
	for _, row := range [][]string{{"a-1", "mem-1"}, {"a-2", "mem-2"}} {
		if err := db.Exec(insert, row[0], row[1]).Error; err != nil {
			t.Fatalf("seed answer: %v", err)
		}
	}
	if err := m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	listed, err := m.ListAllAnswers(ctx, "f-1", "", "")
	if err != nil {
		t.Fatalf("ListAllAnswers: %v", err)
	}
	if len(listed) != 1 || listed[0].MemberID != "mem-2" {
		t.Fatalf("listed answers = %+v, want only mem-2's", listed)
	}
	var kept int64
	if err := db.Raw("select count(*) from " + table + " where member_id = 'mem-1' and withdrawn_at <> ''").Scan(&kept).Error; err != nil || kept != 1 {
		t.Fatalf("withdrawn rows kept = %d (%v), want 1", kept, err)
	}
}

func TestHidingAnswersOfNobodyIsRefused(t *testing.T) {
	if err := gatedManager(t, nil).HideMemberAnswers(context.Background(), ""); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("HideMemberAnswers empty = %v, want ErrInvalidAnswer", err)
	}
}

func TestAWithdrawnMembersAnswersLeaveTheReportCountButStayStoredAndRestorable(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	answerObject, _ := s.Object("answer")
	submissionObject, _ := s.Object("submission")
	answers, submissions := s.TableFor(answerObject), s.TableFor(submissionObject)
	for _, member := range []string{"mem-1", "mem-2"} {
		err := db.Exec("insert into "+submissions+" (id, form_id, member_id, submitted_at) values (?, 'f-1', ?, '2026-01-01T00:00:00Z')", "s-"+member, member).Error
		if err != nil {
			t.Fatalf("seed submission: %v", err)
		}
		for _, question := range []string{"q-1", "q-2"} {
			err := db.Exec("insert into "+answers+" (id, form_id, question_id, member_id, value_text, answered_at) values (?, 'f-1', ?, ?, 'yes', '2026-01-01T00:00:00Z')", "a-"+member+question, question, member).Error
			if err != nil {
				t.Fatalf("seed answer: %v", err)
			}
		}
	}
	reported := func() map[string]int {
		listed, err := m.ListSubmissions(ctx, "f-1", "", "", "")
		if err != nil {
			t.Fatalf("ListSubmissions: %v", err)
		}
		counts := map[string]int{}
		for _, sub := range listed {
			counts[sub.MemberID] = sub.Answers
		}
		return counts
	}
	if got := reported(); got["mem-1"] != 2 || got["mem-2"] != 2 {
		t.Fatalf("counts before withdrawal = %v, want 2 and 2", got)
	}
	if err := m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if got := reported(); len(got) != 1 || got["mem-2"] != 2 {
		t.Fatalf("counts after withdrawal = %v, want mem-2's 2 alone", got)
	}
	var stored int64
	db.Raw("select count(*) from " + answers + " where member_id = 'mem-1'").Scan(&stored)
	if stored != 2 {
		t.Fatalf("stored answers of the withdrawn member = %d, want 2", stored)
	}
	if err := m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers: %v", err)
	}
	if got := reported(); got["mem-1"] != 2 {
		t.Fatalf("count after restore = %v, want 2", got)
	}
	listed, err := m.ListAllAnswers(ctx, "f-1", "mem-1", "")
	if err != nil || len(listed) != 2 || listed[0].ValueText != "yes" {
		t.Fatalf("restored answers = %+v (%v), want both back with their text", listed, err)
	}
}

func seedAnswersOfTwoMembers(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	insert := fmt.Sprintf("insert into %s (id, form_id, question_id, member_id, value_text, answered_at) values (?, 'f-1', 'q-1', ?, ?, '2026-01-01T00:00:00Z')", table)
	for _, row := range [][]string{{"a-1", "mem-1", "yes"}, {"a-2", "mem-2", "no"}} {
		if err := db.Exec(insert, row[0], row[1], row[2]).Error; err != nil {
			t.Fatalf("seed answer: %v", err)
		}
	}
}

func TestRestoringAMembersAnswersListsThemAgain(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	o, _ := s.Object("answer")
	seedAnswersOfTwoMembers(t, db, s.TableFor(o))
	if err := m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if err := m.RestoreMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("RestoreMemberAnswers: %v", err)
	}
	listed, err := m.ListAllAnswers(ctx, "f-1", "mem-1", "")
	if err != nil || len(listed) != 1 || listed[0].ValueText != "yes" || listed[0].WithdrawnAt != "" {
		t.Fatalf("answers after restore = %+v (%v), want mem-1's answer back and unmarked", listed, err)
	}
	if err := m.RestoreMemberAnswers(ctx, ""); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("RestoreMemberAnswers empty = %v, want ErrInvalidAnswer", err)
	}
}

func TestListingHiddenAnswersNamesOnlyThatMembersWithdrawnOnes(t *testing.T) {
	ctx := context.Background()
	m, db, s := gatedManagerOn(t, nil)
	o, _ := s.Object("answer")
	seedAnswersOfTwoMembers(t, db, s.TableFor(o))
	if hidden, err := m.ListHiddenAnswers(ctx, "mem-1"); err != nil || len(hidden) != 0 {
		t.Fatalf("hidden before withdrawal = %+v (%v), want none", hidden, err)
	}
	if err := m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	hidden, err := m.ListHiddenAnswers(ctx, "mem-1")
	if err != nil || len(hidden) != 1 || hidden[0].MemberID != "mem-1" || hidden[0].ValueText != "yes" || hidden[0].WithdrawnAt == "" {
		t.Fatalf("hidden answers = %+v (%v), want mem-1's answer with its text and a stamp", hidden, err)
	}
	if other, err := m.ListHiddenAnswers(ctx, "mem-2"); err != nil || len(other) != 0 {
		t.Fatalf("hidden answers of a member who did not withdraw = %+v (%v), want none", other, err)
	}
	if _, err := m.ListHiddenAnswers(ctx, ""); !errors.Is(err, forms.ErrInvalidAnswer) {
		t.Fatalf("ListHiddenAnswers empty = %v, want ErrInvalidAnswer", err)
	}
}
