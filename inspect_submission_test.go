package mwanachamaforms_test

import (
	"context"
	"errors"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"testing"
)

func councilRowOf(t *testing.T, r voiceRound) models.Submission {
	t.Helper()
	rows, err := r.m.ListSubmissions(context.Background(), r.form.ID, "", "", "")
	if err != nil || len(rows) != 1 {
		t.Fatalf("council list = %+v (%v), want one send", rows, err)
	}
	return rows[0]
}
func TestRecordingAFindingSetsTheFourFieldsAndTheCouncilListShowsThem(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	before := models.NowRFC3339()
	got, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionDoesNotComply, "fence is 2.4 m on the street side", "inspector-1")
	if err != nil {
		t.Fatalf("InspectSubmission: %v", err)
	}
	if got.InspectionResult != models.InspectionDoesNotComply || got.InspectionNote != "fence is 2.4 m on the street side" || got.InspectedBy != "inspector-1" || got.InspectedAt < before {
		t.Fatalf("inspected send = %+v", got)
	}
	listed := councilRowOf(t, r)
	if listed.InspectionResult != got.InspectionResult || listed.InspectionNote != got.InspectionNote || listed.InspectedBy != got.InspectedBy || listed.InspectedAt != got.InspectedAt {
		t.Fatalf("council row = %+v, want the finding as recorded %+v", listed, got)
	}
}
func TestTheInspectionResultWordsAreFixed(t *testing.T) {
	if models.InspectionComplies != "complies" || models.InspectionDoesNotComply != "does_not_comply" || models.InspectionNeedsMore != "needs_more" {
		t.Fatalf("words = %q %q %q", models.InspectionComplies, models.InspectionDoesNotComply, models.InspectionNeedsMore)
	}
}
func TestTheFindingNoteIsTrimmed(t *testing.T) {
	r := sentOnce(t)
	got, err := r.m.InspectSubmission(context.Background(), r.form.ID, "mem-1", models.InspectionNeedsMore, "  a site plan is missing \n", "inspector-1")
	if err != nil || got.InspectionNote != "a site plan is missing" {
		t.Fatalf("trimmed note = %+v (%v)", got, err)
	}
}
func TestInspectingAgainReplacesTheEarlierFinding(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionDoesNotComply, "plan missing", "inspector-1"); err != nil {
		t.Fatalf("first finding: %v", err)
	}
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionComplies, "", "inspector-2"); err != nil {
		t.Fatalf("second finding: %v", err)
	}
	listed := councilRowOf(t, r)
	if listed.InspectionResult != models.InspectionComplies || listed.InspectedBy != "inspector-2" || listed.InspectionNote != "" {
		t.Fatalf("send after inspecting again = %+v, want the second finding alone", listed)
	}
}
func TestACompliesFindingNeedsNoNote(t *testing.T) {
	r := sentOnce(t)
	got, err := r.m.InspectSubmission(context.Background(), r.form.ID, "mem-1", models.InspectionComplies, "", "inspector-1")
	if err != nil || got.InspectionResult != models.InspectionComplies || got.InspectionNote != "" {
		t.Fatalf("complies without note = %+v (%v)", got, err)
	}
}
func TestAFindingOtherThanCompliesWithoutANoteIsRefusedAndLeavesTheSendUnchanged(t *testing.T) {
	ctx := context.Background()
	for _, result := range []models.InspectionResult{models.InspectionDoesNotComply, models.InspectionNeedsMore} {
		for _, note := range []string{"", "   "} {
			r := sentOnce(t)
			if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", result, note, "inspector-1"); !errors.Is(err, forms.ErrMissingNote) {
				t.Fatalf("%s with note %q = %v, want ErrMissingNote", result, note, err)
			}
			listed := councilRowOf(t, r)
			if listed.InspectionResult != "" || listed.InspectedBy != "" || listed.InspectedAt != "" || listed.InspectionNote != "" {
				t.Fatalf("send after the refused %s = %+v, want no finding", result, listed)
			}
		}
	}
}
func TestAnUnknownResultIsAnInvalidInspection(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	for _, word := range []string{"", "maybe", "Complies", "approved"} {
		if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionResult(word), "n", "inspector-1"); !errors.Is(err, forms.ErrInvalidInspection) {
			t.Fatalf("result %q = %v, want ErrInvalidInspection", word, err)
		}
	}
}
func TestAnEmptyInspectorIsAnInvalidInspection(t *testing.T) {
	r := sentOnce(t)
	if _, err := r.m.InspectSubmission(context.Background(), r.form.ID, "mem-1", models.InspectionComplies, "", ""); !errors.Is(err, forms.ErrInvalidInspection) {
		t.Fatalf("nobody inspecting = %v, want ErrInvalidInspection", err)
	}
}
func TestTheApplicantCannotInspectTheirOwnSend(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionComplies, "", "mem-1"); !errors.Is(err, forms.ErrSelfInspection) {
		t.Fatalf("own send inspected by the applicant = %v, want ErrSelfInspection", err)
	}
	if listed := councilRowOf(t, r); listed.InspectionResult != "" {
		t.Fatalf("send after the refused self inspection = %+v, want no finding", listed)
	}
}
func TestInspectingAnUnknownSendIsNotFound(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-9", models.InspectionComplies, "", "inspector-1"); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("nobody sent = %v, want ErrSubmissionNotFound", err)
	}
	if _, err := r.m.InspectSubmission(ctx, "no-such-form", "mem-1", models.InspectionComplies, "", "inspector-1"); err == nil {
		t.Fatal("unknown form inspected without error")
	}
}
func TestInspectingAWithdrawnSendIsNotFound(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionComplies, "", "inspector-1"); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("withdrawn send = %v, want ErrSubmissionNotFound", err)
	}
}
func TestADecisionStaysAfterAnInspection(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "ok", "clerk-1"); err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionNeedsMore, "photo of the gate", "inspector-1"); err != nil {
		t.Fatalf("InspectSubmission: %v", err)
	}
	listed := councilRowOf(t, r)
	if listed.Decision != models.DecisionApproved || listed.DecidedBy != "clerk-1" || listed.DecisionNote != "ok" || listed.InspectionResult != models.InspectionNeedsMore {
		t.Fatalf("send after deciding then inspecting = %+v", listed)
	}
}
func TestAFindingStaysAfterADecision(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionDoesNotComply, "too tall", "inspector-1"); err != nil {
		t.Fatalf("InspectSubmission: %v", err)
	}
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionRefused, "too tall", "clerk-1"); err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	listed := councilRowOf(t, r)
	if listed.InspectionResult != models.InspectionDoesNotComply || listed.InspectionNote != "too tall" || listed.InspectedBy != "inspector-1" || listed.Decision != models.DecisionRefused {
		t.Fatalf("send after inspecting then deciding = %+v", listed)
	}
}
func TestTheApplicantsOwnReadsCarryNoFinding(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.InspectSubmission(ctx, r.form.ID, "mem-1", models.InspectionDoesNotComply, "too tall", "inspector-1"); err != nil {
		t.Fatalf("InspectSubmission: %v", err)
	}
	mine, err := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if err != nil {
		t.Fatalf("ShowMySubmission: %v", err)
	}
	if mine.InspectionResult != "" || mine.InspectionNote != "" || mine.InspectedBy != "" || mine.InspectedAt != "" {
		t.Fatalf("read_mine = %+v, want no finding", mine)
	}
	rows, err := r.m.ListAnswered(ctx, "mem-1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListAnswered = %+v (%v)", rows, err)
	}
	send, err := r.m.GetSend(ctx, r.form.ID, rows[0].SendID, "mem-1")
	if err != nil {
		t.Fatalf("GetSend: %v", err)
	}
	got := send.Submission
	if got.InspectionResult != "" || got.InspectionNote != "" || got.InspectedBy != "" || got.InspectedAt != "" {
		t.Fatalf("read_my_send submission = %+v, want no finding", got)
	}
}
