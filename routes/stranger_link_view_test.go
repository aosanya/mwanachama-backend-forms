package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAStrangerOpeningALinkReadsNoStaffMemberId(t *testing.T) {
	x := newLinkFixture(t)
	rec := doAs(t, x.mux, "", http.MethodGet, "/public-links/"+x.linked.key, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("open_public_link = %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "sender-1") {
		t.Fatalf("a stranger reads the publisher's member id: %s", body)
	}
	if !strings.Contains(body, "Street lighting") {
		t.Fatalf("the respondent view lost the form title: %s", body)
	}
}
