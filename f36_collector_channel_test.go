package mwanachamaforms_test

import (
	"context"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestASendInTheAppNamesTheSenderAsItsCollectorOnTheAppChannel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Collected"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	first, err := mgr.GetSubmission(ctx, f.ID, "alice")
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if first.Collector != "alice" || first.Channel != models.ChannelApp || first.CollectedAt != first.SubmittedAt {
		t.Fatalf("first send = %+v, want alice collecting on the app at the send", first)
	}
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatalf("resend: %v", err)
	}
	again, _ := mgr.GetSubmission(ctx, f.ID, "alice")
	if again.CollectedAt != again.EditedAt || again.Channel != models.ChannelApp || again.Collector != "alice" {
		t.Fatalf("resend = %+v, want the collection moved to the resend", again)
	}
}
