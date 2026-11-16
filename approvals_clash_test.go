package mwanachamaforms_test

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type agencyApproval struct {
	ID           string `gorm:"primaryKey"`
	Kind         string
	Summary      string
	Payload      string
	PayloadPatch string
	Status       string
	ProposedBy   string
	AmendedBy    string
	AmendedAt    string
	Amendments   string
	DecidedBy    string
	Reason       string
	ProposedAt   string
	DecidedAt    string
	Applied      string
}

type memoryApprovalStore struct{}

func (memoryApprovalStore) TableFor(agency string) string {
	return agency + "_approvals"
}

func (s memoryApprovalStore) Open(db *gorm.DB, agency string) error {
	return db.Table(s.TableFor(agency)).AutoMigrate(&agencyApproval{})
}

var approvalStore forms.AgencyApprovals = memoryApprovalStore{}

func provisionWithStore(t *testing.T, db *gorm.DB, s *spec.Spec) {
	t.Helper()
	if err := forms.Provision(db, s, forms.WithAgencyApprovals(approvalStore)); err != nil {
		t.Fatalf("Provision: %v", err)
	}
}

func proposeApprovals(t *testing.T, db *gorm.DB, summaries ...string) []agencyApproval {
	t.Helper()
	if err := approvalStore.Open(db, clashSlug); err != nil {
		t.Fatalf("open the agency approvals store: %v", err)
	}
	var made []agencyApproval
	for _, summary := range summaries {
		a := agencyApproval{ID: "approval-" + summary, Kind: "instance_rename", Summary: summary, Payload: `{"name":"Z"}`, Status: "pending", ProposedBy: "owner-1"}
		if err := db.Table(approvalStore.TableFor(clashSlug)).Create(&a).Error; err != nil {
			t.Fatalf("propose %q: %v", summary, err)
		}
		made = append(made, a)
	}
	return made
}

func listedSummaries(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var listed []agencyApproval
	if err := db.Table(approvalStore.TableFor(clashSlug)).Find(&listed).Error; err != nil {
		t.Fatalf("list the agency approvals: %v", err)
	}
	out := make([]string, 0, len(listed))
	for _, a := range listed {
		out = append(out, a.Summary)
	}
	return out
}

func formsApprovalTable(s *spec.Spec) string {
	o, _ := s.ByRole("approval")
	return s.TableFor(o)
}

func sameSummaries(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, g := range got {
		seen[g]++
	}
	for _, w := range want {
		seen[w]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func swallowedByForms(t *testing.T, db *gorm.DB, s *spec.Spec, summaries ...string) {
	t.Helper()
	provisionWithStore(t, db, s)
	proposeApprovals(t, db, summaries...)
	agency, swallowed := approvalStore.TableFor(clashSlug), formsApprovalTable(s)
	types, err := db.Migrator().ColumnTypes(agency)
	if err != nil {
		t.Fatalf("ColumnTypes: %v", err)
	}
	var names []string
	for _, c := range types {
		names = append(names, c.Name())
		if db.Migrator().HasColumn(swallowed, c.Name()) {
			continue
		}
		if err := db.Exec(fmt.Sprintf("alter table %s add column %s text", swallowed, c.Name())).Error; err != nil {
			t.Fatalf("add column %s: %v", c.Name(), err)
		}
	}
	list := strings.Join(names, ", ")
	if err := db.Exec(fmt.Sprintf("insert into %s (%s, form_id) select %s, '' from %s", swallowed, list, list, agency)).Error; err != nil {
		t.Fatalf("copy: %v", err)
	}
	if err := db.Exec("drop table " + agency).Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}
}

func TestFormsUsedThenAnApprovalProposedThenARestartStillProvisions(t *testing.T) {
	db, s := clashDB(t)
	provisionWithStore(t, db, s)
	proposeApprovals(t, db, "rename")

	provisionWithStore(t, db, s)

	if got := listedSummaries(t, db); !sameSummaries(got, []string{"rename"}) {
		t.Fatalf("approvals after the restart = %v, want [rename]", got)
	}
}

func TestAnApprovalProposedBeforeTheFirstFormsCallIsLeftWhereItIs(t *testing.T) {
	db, s := clashDB(t)
	proposeApprovals(t, db, "rename", "second")

	provisionWithStore(t, db, s)

	if got := listedSummaries(t, db); !sameSummaries(got, []string{"rename", "second"}) {
		t.Fatalf("approvals after forms were provisioned = %v, want [rename second]", got)
	}
	if n := rowsIn(t, db, formsApprovalTable(s)); n != 0 {
		t.Fatalf("the forms approvals table holds %d row(s) of the agency store", n)
	}
}

func TestAnApprovalProposedBeforeTheFirstFormsCallNeedsNoStoreToBeLeftAlone(t *testing.T) {
	db, s := clashDB(t)
	proposeApprovals(t, db, "rename")

	mustProvision(t, db, s)

	if got := listedSummaries(t, db); !sameSummaries(got, []string{"rename"}) {
		t.Fatalf("approvals after forms were provisioned = %v, want [rename]", got)
	}
}

func TestApprovalsAlreadySwallowedByFormsAreGivenBack(t *testing.T) {
	db, s := clashDB(t)
	swallowedByForms(t, db, s, "rename", "second")

	provisionWithStore(t, db, s)

	if got := listedSummaries(t, db); !sameSummaries(got, []string{"rename", "second"}) {
		t.Fatalf("approvals after the repair = %v, want [rename second]", got)
	}
	if n := rowsIn(t, db, formsApprovalTable(s)); n != 0 {
		t.Fatalf("the forms approvals table still holds %d row(s) of the agency store", n)
	}
}

func TestProvisionFailsOnSwallowedApprovalsWhenNoStoreWasPassed(t *testing.T) {
	db, s := clashDB(t)
	swallowedByForms(t, db, s, "rename")

	err := forms.Provision(db, s)

	if err == nil {
		t.Fatalf("Provision skipped the swallowed approvals without a store")
	}
	if !strings.Contains(err.Error(), "agency approvals") {
		t.Fatalf("error = %v, want it to name the agency approvals", err)
	}
	if n := rowsIn(t, db, formsApprovalTable(s)); n != 1 {
		t.Fatalf("the forms approvals table holds %d row(s), want the swallowed one left in place", n)
	}
}

func TestTheRepairKeepsApprovalsProposedAfterTheDamageAndFormsRoundsAlike(t *testing.T) {
	db, s := clashDB(t)
	swallowedByForms(t, db, s, "old one")
	proposeApprovals(t, db, "new one")
	formsRound := fmt.Sprintf("insert into %s (id, form_id, requested_by, requested_at, decided_at) values ('r-1', 'f-1', 'm-1', '2026-01-01T00:00:00Z', '')", formsApprovalTable(s))
	if err := db.Exec(formsRound).Error; err != nil {
		t.Fatalf("insert forms round: %v", err)
	}

	provisionWithStore(t, db, s)
	provisionWithStore(t, db, s)

	if got := listedSummaries(t, db); !sameSummaries(got, []string{"old one", "new one"}) {
		t.Fatalf("approvals after the repair = %v, want [old one new one]", got)
	}
	if n := rowsIn(t, db, formsApprovalTable(s)); n != 1 {
		t.Fatalf("the forms approvals table holds %d row(s), want its one forms round", n)
	}
}

func TestTheRepairNeverOverwritesAnApprovalThatAlreadyHasThatId(t *testing.T) {
	db, s := clashDB(t)
	swallowedByForms(t, db, s, "kept in forms table")
	var swallowed agencyApproval
	if err := db.Table(formsApprovalTable(s)).Where("kind <> ''").Take(&swallowed).Error; err != nil {
		t.Fatalf("read swallowed row: %v", err)
	}
	if err := approvalStore.Open(db, clashSlug); err != nil {
		t.Fatalf("open the agency approvals store: %v", err)
	}
	twin := swallowed
	twin.Summary = "already given back"
	if err := db.Table(approvalStore.TableFor(clashSlug)).Create(&twin).Error; err != nil {
		t.Fatalf("create twin: %v", err)
	}

	provisionWithStore(t, db, s)

	var got agencyApproval
	if err := db.Table(approvalStore.TableFor(clashSlug)).Where("id = ?", swallowed.ID).Take(&got).Error; err != nil {
		t.Fatalf("read the store's row: %v", err)
	}
	if got.Summary != "already given back" {
		t.Fatalf("the repair overwrote the store's row: summary = %q", got.Summary)
	}
	if n := rowsIn(t, db, formsApprovalTable(s)); n != 1 {
		t.Fatalf("the forms approvals table holds %d row(s), want the differing copy kept", n)
	}
}

func TestALegacyFormsApprovalsTableOfThatNameIsStillRenamedIntoForms(t *testing.T) {
	db, s := clashDB(t)
	legacy := approvalStore.TableFor(clashSlug)
	create := fmt.Sprintf("create table %s (id text primary key, form_id text not null, requested_by text, requested_at text, decided_by text, decided_at text, decision text, note text, slot text)", legacy)
	insert := fmt.Sprintf("insert into %s (id, form_id, requested_by, requested_at, decided_at) values ('r-1', 'f-1', 'm-1', '2026-01-01T00:00:00Z', '')", legacy)
	for _, stmt := range []string{create, insert} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("legacy table: %v", err)
		}
	}

	provisionWithStore(t, db, s)

	if n := rowsIn(t, db, formsApprovalTable(s)); n != 1 {
		t.Fatalf("the forms approvals table holds %d row(s), want the legacy round", n)
	}
	if db.Migrator().HasTable(legacy) {
		t.Fatalf("the legacy forms table was left behind")
	}
}
