package mwanachamaforms_test

import (
	"context"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"testing"
)

func sendNamingARow(t *testing.T, r voiceRound, member, recordRowID string) {
	t.Helper()
	_, err := r.m.SubmitAnswersFor(context.Background(), r.form.ID, member, "g-1", "", "", recordRowID, []models.Answer{{QuestionID: r.question.ID, ValueText: "market stall licence"}, {QuestionID: r.reason.ID, ValueText: "new stall"}})
	if err != nil {
		t.Fatalf("send naming row %q: %v", recordRowID, err)
	}
}
func TestASendKeepsTheRecordRowItNamesAndItIsListedForThatRow(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	sendNamingARow(t, r, "mem-1", "row-1")
	listed, err := r.m.ListSubmissionsOfPerson(ctx, "mem-1", "row-1")
	if err != nil || len(listed) != 1 {
		t.Fatalf("ListSubmissionsOfPerson row-1 = %+v (%v), want the one send", listed, err)
	}
	if listed[0].RecordRowID != "row-1" {
		t.Fatalf("listed send record_row_id = %q, want row-1", listed[0].RecordRowID)
	}
	byForm, err := r.m.ListSubmissions(ctx, r.form.ID, "mem-1", "", "row-1")
	if err != nil || len(byForm) != 1 || byForm[0].RecordRowID != "row-1" {
		t.Fatalf("ListSubmissions row-1 = %+v (%v), want the one send", byForm, err)
	}
}
func TestASendIsNotListedForAnotherRow(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	sendNamingARow(t, r, "mem-1", "row-1")
	listed, err := r.m.ListSubmissionsOfPerson(ctx, "mem-1", "row-2")
	if err != nil || len(listed) != 0 {
		t.Fatalf("ListSubmissionsOfPerson row-2 = %+v (%v), want none", listed, err)
	}
	byForm, err := r.m.ListSubmissions(ctx, r.form.ID, "mem-1", "", "row-2")
	if err != nil || len(byForm) != 0 {
		t.Fatalf("ListSubmissions row-2 = %+v (%v), want none", byForm, err)
	}
}
func TestASendThatNamesNoRowIsNeverListedForARow(t *testing.T) {
	ctx := context.Background()
	r := openRoundForTwoGroups(t, gatedManager(t, nil), models.OneVoiceOff)
	sendNamingARow(t, r, "mem-1", "")
	listed, err := r.m.ListSubmissionsOfPerson(ctx, "mem-1", "row-1")
	if err != nil || len(listed) != 0 {
		t.Fatalf("ListSubmissionsOfPerson row-1 = %+v (%v), want none for a send naming no row", listed, err)
	}
	everything, err := r.m.ListSubmissionsOfPerson(ctx, "mem-1", "")
	if err != nil || len(everything) != 1 {
		t.Fatalf("ListSubmissionsOfPerson without a row = %+v (%v), want the send", everything, err)
	}
}
