package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAResidentReadsAFormWithoutTheMembersWhoPublishedIt(t *testing.T) {
	fm, sent, _, _ := residentFixture(t)

	resident := doAs(t, mountReachedOnly(t, fm, true), "alice", http.MethodGet, "/forms/"+sent.form.ID, "")
	if resident.Code != http.StatusOK {
		t.Fatalf("get_form as a resident = %d %s", resident.Code, resident.Body.String())
	}
	if strings.Contains(resident.Body.String(), "sender-1") {
		t.Fatalf("a resident reads the publisher's member id: %s", resident.Body.String())
	}

	staff := doAs(t, mountReachedOnly(t, fm, false), "alice", http.MethodGet, "/forms/"+sent.form.ID, "")
	if !strings.Contains(staff.Body.String(), "sender-1") {
		t.Fatalf("staff lost the whole record: %s", staff.Body.String())
	}
}

func TestAResidentsWaitingListNamesNoPublishingMember(t *testing.T) {
	fm, _, _, _ := residentFixture(t)
	rec := callAs(t, mountAsCaller(t, fm), http.MethodGet, "/forms/waiting", "alice", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list_waiting = %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sender-1") || strings.Contains(rec.Body.String(), "published_by") {
		t.Fatalf("the waiting list names the publishing member: %s", rec.Body.String())
	}
}
