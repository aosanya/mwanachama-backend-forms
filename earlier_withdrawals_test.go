package mwanachamaforms_test

import (
	"testing"

	"gorm.io/gorm"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	withdrewAt   = "2026-03-01T10:00:00.000000000Z"
	withdrewTwo  = "2026-04-01T10:00:00.000000000Z"
	earlierStamp = "2026-02-01T09:00:00.000000000Z"
)

func tableOf(t *testing.T, s *spec.Spec, role string) string {
	t.Helper()
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("no object with role %q", role)
	}
	return s.TableFor(o)
}

func insertOldSend(t *testing.T, db *gorm.DB, s *spec.Spec, id, member, withdrawnAt string) {
	t.Helper()
	stmt := "insert into " + tableOf(t, s, "submission") + " (id, form_id, member_id, submitted_at, withdrawn_at) values (?, 'f-1', ?, '2026-01-01T00:00:00Z', ?)"
	if err := db.Exec(stmt, id, member, withdrawnAt).Error; err != nil {
		t.Fatalf("insert send %s: %v", id, err)
	}
}

func insertOldAnswer(t *testing.T, db *gorm.DB, s *spec.Spec, id, member, withdrawnAt string) {
	t.Helper()
	stmt := "insert into " + tableOf(t, s, "answer") + " (id, form_id, question_id, member_id, value_text, answered_at, withdrawn_at) values (?, 'f-1', 'q-1', ?, 'yes', '2026-01-01T00:00:00Z', ?)"
	if err := db.Exec(stmt, id, member, withdrawnAt).Error; err != nil {
		t.Fatalf("insert answer %s: %v", id, err)
	}
}

func stampOf(t *testing.T, db *gorm.DB, s *spec.Spec, role, id string) string {
	t.Helper()
	var stamp string
	if err := db.Raw("select withdrawn_at from "+tableOf(t, s, role)+" where id = ?", id).Scan(&stamp).Error; err != nil {
		t.Fatalf("read stamp of %s: %v", id, err)
	}
	return stamp
}

func theOldState(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	db, s := clashDB(t)
	mustProvision(t, db, s)
	insertOldAnswer(t, db, s, "ans-gone", "mem-gone", earlierStamp)
	insertOldSend(t, db, s, "send-gone", "mem-gone", "")
	insertOldAnswer(t, db, s, "ans-live", "mem-live", "")
	insertOldSend(t, db, s, "send-live", "mem-live", "")
	return db, s
}

func TestTheSendsOfAMemberWhoWithdrewBeforeTheyWereHiddenAreStampedWithTheWithdrawalTime(t *testing.T) {
	db, s := theOldState(t)

	got, err := forms.StampEarlierWithdrawals(db, s, []forms.EarlierWithdrawal{{MemberID: "mem-gone", WithdrawnAt: withdrewAt}})
	if err != nil {
		t.Fatalf("StampEarlierWithdrawals: %v", err)
	}

	if stamp := stampOf(t, db, s, "submission", "send-gone"); stamp != withdrewAt {
		t.Errorf("the withdrawn member's send is stamped %q, want the withdrawal time %q", stamp, withdrewAt)
	}
	if stamp := stampOf(t, db, s, "answer", "ans-gone"); stamp != earlierStamp {
		t.Errorf("the withdrawn member's hidden answer is stamped %q, want it left at %q", stamp, earlierStamp)
	}
	if got.Members != 1 || got.Sends != 1 || got.Answers != 0 {
		t.Errorf("counts = %+v, want 1 member, 1 send, 0 answers", got)
	}
}

func TestTheStampTouchesNoRowOfAMemberWhoWasNotNamed(t *testing.T) {
	db, s := theOldState(t)

	if _, err := forms.StampEarlierWithdrawals(db, s, []forms.EarlierWithdrawal{{MemberID: "mem-gone", WithdrawnAt: withdrewAt}}); err != nil {
		t.Fatalf("StampEarlierWithdrawals: %v", err)
	}

	for _, role := range []string{"submission", "answer"} {
		id := map[string]string{"submission": "send-live", "answer": "ans-live"}[role]
		if stamp := stampOf(t, db, s, role, id); stamp != "" {
			t.Errorf("the %s of a member whose consent is live is stamped %q, want it untouched", role, stamp)
		}
	}
}

func TestTheStampRunTwiceChangesNothingTheSecondTime(t *testing.T) {
	db, s := theOldState(t)
	named := []forms.EarlierWithdrawal{{MemberID: "mem-gone", WithdrawnAt: withdrewAt}}
	if _, err := forms.StampEarlierWithdrawals(db, s, named); err != nil {
		t.Fatalf("first run: %v", err)
	}
	before := rowsIn(t, db, tableOf(t, s, "submission"))

	again, err := forms.StampEarlierWithdrawals(db, s, []forms.EarlierWithdrawal{{MemberID: "mem-gone", WithdrawnAt: withdrewTwo}})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	if again != (forms.EarlierWithdrawalsStamped{}) {
		t.Errorf("second run counts = %+v, want nothing stamped", again)
	}
	if stamp := stampOf(t, db, s, "submission", "send-gone"); stamp != withdrewAt {
		t.Errorf("the second run moved the stamp to %q, want %q kept", stamp, withdrewAt)
	}
	if after := rowsIn(t, db, tableOf(t, s, "submission")); after != before {
		t.Errorf("sends held = %d after the second run, want %d: no row is deleted or added", after, before)
	}
}

func TestAnEarlierHiddenSendOfTheSameMemberIsLeftAtItsOwnStamp(t *testing.T) {
	db, s := clashDB(t)
	mustProvision(t, db, s)
	insertOldSend(t, db, s, "send-old", "mem-gone", earlierStamp)
	insertOldSend(t, db, s, "send-new", "mem-gone", "")

	if _, err := forms.StampEarlierWithdrawals(db, s, []forms.EarlierWithdrawal{{MemberID: "mem-gone", WithdrawnAt: withdrewAt}}); err != nil {
		t.Fatalf("StampEarlierWithdrawals: %v", err)
	}

	if stamp := stampOf(t, db, s, "submission", "send-old"); stamp != earlierStamp {
		t.Errorf("the send hidden at an earlier withdrawal is stamped %q, want %q kept", stamp, earlierStamp)
	}
	if stamp := stampOf(t, db, s, "submission", "send-new"); stamp != withdrewAt {
		t.Errorf("the live send is stamped %q, want %q", stamp, withdrewAt)
	}
}

func TestAFailureHalfwayLeavesEveryRowAsItWasAndReturnsTheError(t *testing.T) {
	db, s := theOldState(t)
	insertOldSend(t, db, s, "send-second", "mem-second", "")
	table := tableOf(t, s, "submission")
	trigger := "create trigger refuse_second before update on " + table +
		" when new.member_id = 'mem-second' begin select raise(abort, 'refused'); end"
	if err := db.Exec(trigger).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	got, err := forms.StampEarlierWithdrawals(db, s, []forms.EarlierWithdrawal{
		{MemberID: "mem-gone", WithdrawnAt: withdrewAt},
		{MemberID: "mem-second", WithdrawnAt: withdrewAt},
	})

	if err == nil {
		t.Fatal("StampEarlierWithdrawals = nil, want the failure")
	}
	if got != (forms.EarlierWithdrawalsStamped{}) {
		t.Errorf("counts after a failure = %+v, want nothing reported", got)
	}
	for _, id := range []string{"send-gone", "send-second"} {
		if stamp := stampOf(t, db, s, "submission", id); stamp != "" {
			t.Errorf("%s is stamped %q after a failed run, want it as it was", id, stamp)
		}
	}
}

func TestAnEntryWithNoMemberOrNoTimeStampsNothing(t *testing.T) {
	db, s := theOldState(t)

	got, err := forms.StampEarlierWithdrawals(db, s, []forms.EarlierWithdrawal{{MemberID: "", WithdrawnAt: withdrewAt}, {MemberID: "mem-gone", WithdrawnAt: ""}})
	if err != nil {
		t.Fatalf("StampEarlierWithdrawals: %v", err)
	}

	if got != (forms.EarlierWithdrawalsStamped{}) || stampOf(t, db, s, "submission", "send-gone") != "" {
		t.Errorf("an entry without a member or a time stamped %+v, want nothing", got)
	}
}
