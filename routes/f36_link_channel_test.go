package routes_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestF36ASendThroughAClaimedLinkIsStampedOnTheLinkChannel(t *testing.T) {
	x := newLinkFixture(t)
	x.registerThroughLink(t, x.linked.key, "token-c1")
	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("claim_respondent = %d, want 200", res.StatusCode)
	}
	assertOpen(t, x, "carol", "the linked form", x.linked.choiceForm)

	sent, err := x.fm.GetSubmission(context.Background(), x.linked.form.ID, "carol")
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if sent.Channel != models.ChannelLink || sent.Collector != "carol" || sent.CollectedAt == "" {
		t.Fatalf("send = %+v, want carol collecting on the link channel", sent)
	}
}
