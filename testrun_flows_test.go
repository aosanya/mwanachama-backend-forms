package mwanachamaforms_test

import (
	"context"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func TestARunThroughEveryObjectLeavesNoRowInAnyLiveTableAndEveryRowCarriesTheRun(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	s := testSpec(t, "test")
	ctx := context.Background()

	mgr.UseApprovalRequired(true)
	approving := forRun(t, mgr, "run-a")
	draft := newDraftForm(t, approving)
	if _, err := approving.Submit(ctx, draft.ID, "author"); err != nil {
		t.Fatal(err)
	}
	if _, err := approving.Approve(ctx, draft.ID, "reviewer", ""); err != nil {
		t.Fatal(err)
	}

	mgr.UseApprovalRequired(false)
	run := forRun(t, mgr, "run-a")
	choice, err := run.Create(ctx, mwanachamaforms.Form{Title: "Choice", OriginatorChapterID: "council", ClosesAt: futureRFC3339(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	q, _, err := run.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: choice.ID, AnswerType: mwanachamaforms.AnswerSingleChoice, Prompt: "Which ward?",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.AddOption(ctx, q.ID, "North", ""); err != nil {
		t.Fatal(err)
	}
	translations, ok := run.(mwanachamaforms.Translator)
	if !ok {
		t.Fatal("a run's form manager does not translate")
	}
	if _, err := translations.PutTranslation(ctx, models.Translation{
		FormID: choice.ID, SubjectRole: models.TranslationOfForm, SubjectID: choice.ID,
		Field: "title", Language: "sw", Text: "Chaguo", Reviewed: true,
	}); err != nil {
		t.Fatal(err)
	}

	newOpenPublicForm(t, run, "open-key")
	respondent, err := run.RegisterRespondent(ctx, "open-key", "pk-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.DeclareChapter(ctx, "open-key", respondent.ID, "", "my street"); err != nil {
		t.Fatal(err)
	}
	if _, err := run.ClaimRespondent(ctx, "open-key", "pk-1", "alice"); err != nil {
		t.Fatal(err)
	}

	sent, sentQuestion := composeAimedForm(t, run, mwanachamaforms.Form{Title: "Sent"})
	if err := submitOne(run, sent.ID, sentQuestion); err != nil {
		t.Fatal(err)
	}
	if _, err := run.Close(ctx, sent.ID, "closer"); err != nil {
		t.Fatal(err)
	}
	if _, err := run.PublishResults(ctx, sent.ID, "results/sent.md", "abc123", nil, "publisher"); err != nil {
		t.Fatal(err)
	}

	if got := liveRows(t, db, s); got != 0 {
		t.Fatalf("live rows = %d after a run through every object, want 0", got)
	}
	rows, err := specstore.RowsOfRun(ctx, db, s, "run-a")
	if err != nil {
		t.Fatal(err)
	}
	carried := map[string]int64{}
	for _, r := range rows {
		carried[r.Object] = r.Rows
	}
	for _, o := range s.Objects {
		if carried[o.Name] == 0 {
			t.Fatalf("no %s row carries run-a: %+v", o.Name, rows)
		}
	}
	var held int64
	for _, o := range s.Objects {
		var n int64
		if err := db.Table(s.TestTableFor(o)).Where("run_id = ?", "run-a").Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		held += n
	}
	if total := testCopyRows(t, db, s); held != total {
		t.Fatalf("%d of %d test-copy rows carry run-a", held, total)
	}

	if _, err := specstore.RemoveRun(ctx, db, s, "run-a"); err != nil {
		t.Fatal(err)
	}
	if left := testCopyRows(t, db, s); left != 0 {
		t.Fatalf("test copy holds %d rows after the run was removed, want 0", left)
	}
}
