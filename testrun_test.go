package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func liveRows(t *testing.T, db *gorm.DB, s *spec.Spec) int64 {
	t.Helper()
	var total int64
	for _, o := range s.Objects {
		var n int64
		if err := db.Table(s.TableFor(o)).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		total += n
	}
	return total
}

func testCopyRows(t *testing.T, db *gorm.DB, s *spec.Spec) int64 {
	t.Helper()
	var total int64
	for _, o := range s.Objects {
		var n int64
		if err := db.Table(s.TestTableFor(o)).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		total += n
	}
	return total
}

func forRun(t *testing.T, mgr mwanachamaforms.FormManager, id string) mwanachamaforms.FormManager {
	t.Helper()
	run, err := mgr.ForRun(id)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func titles(t *testing.T, mgr mwanachamaforms.FormManager) map[string]bool {
	t.Helper()
	forms, err := mgr.ListForChapter(context.Background(), "council")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, f := range forms {
		out[f.Title] = true
	}
	return out
}

func TestARunThroughASendLeavesNoRowInAnyLiveTable(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	s := testSpec(t, "test")
	run := forRun(t, mgr, "run-a")
	f, q := composeAimedForm(t, run, mwanachamaforms.Form{Title: "Run only"})
	if err := submitOne(run, f.ID, q); err != nil {
		t.Fatal(err)
	}

	if got := liveRows(t, db, s); got != 0 {
		t.Fatalf("live rows = %d after a run, want 0", got)
	}
	rows, err := specstore.RowsOfRun(context.Background(), db, s, "run-a")
	if err != nil {
		t.Fatal(err)
	}
	stored := map[string]int64{}
	for _, r := range rows {
		stored[r.Object] = r.Rows
	}
	for _, object := range []string{"form", "question", "target", "propagation", "answer", "submission"} {
		if stored[object] == 0 {
			t.Fatalf("run-a stored nothing in %s: %+v", object, rows)
		}
	}
}

func TestALiveWriteLeavesTheTestCopyEmpty(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	s := testSpec(t, "test")
	f, q := composeAimedForm(t, mgr, mwanachamaforms.Form{Title: "Live"})
	if err := submitOne(mgr, f.ID, q); err != nil {
		t.Fatal(err)
	}
	if got := testCopyRows(t, db, s); got != 0 {
		t.Fatalf("test copy holds %d rows after a live write, want 0", got)
	}
}

func TestARunsFormIsNotVisibleLiveOrInAnotherRun(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	a, b := forRun(t, mgr, "run-a"), forRun(t, mgr, "run-b")
	made, _ := composeAimedForm(t, a, mwanachamaforms.Form{Title: "Only in A"})

	if !titles(t, a)["Only in A"] {
		t.Fatal("run-a cannot read its own form")
	}
	if titles(t, mgr)["Only in A"] || titles(t, b)["Only in A"] {
		t.Fatal("a run's form is visible outside the run")
	}
	if _, err := mgr.Get(ctx, made.ID); !errors.Is(err, mwanachamaforms.ErrFormNotFound) {
		t.Fatalf("live Get of a run's form = %v, want ErrNotFound", err)
	}
	if _, err := b.Get(ctx, made.ID); !errors.Is(err, mwanachamaforms.ErrFormNotFound) {
		t.Fatalf("run-b Get of run-a's form = %v, want ErrNotFound", err)
	}
}

func TestTwoRunsAndTheLiveCopyKeepTheirSendsApart(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	a, b := forRun(t, mgr, "run-a"), forRun(t, mgr, "run-b")
	fa, qa := composeAimedForm(t, a, mwanachamaforms.Form{Title: "A"})
	composeAimedForm(t, b, mwanachamaforms.Form{Title: "B"})
	if err := submitOne(a, fa.ID, qa); err != nil {
		t.Fatal(err)
	}

	sends, err := a.ListSubmissions(ctx, fa.ID, "", "")
	if err != nil || len(sends) != 1 {
		t.Fatalf("run-a sends = %+v, %v; want one", sends, err)
	}
	for name, other := range map[string]mwanachamaforms.FormManager{"live": mgr, "run-b": b} {
		got, err := other.ListSubmissions(ctx, fa.ID, "", "")
		if err != nil || len(got) != 0 {
			t.Fatalf("%s sees %+v, %v of run-a's sends; want none", name, got, err)
		}
	}
}

func TestADeleteInARunLeavesTheSameKeyInAnotherRun(t *testing.T) {
	mgr, _ := newTestManagerDB(t)
	ctx := context.Background()
	a, b := forRun(t, mgr, "run-a"), forRun(t, mgr, "run-b")
	made, err := a.Create(ctx, mwanachamaforms.Form{Title: "Shared", OriginatorChapterID: "council", ClosesAt: futureRFC3339(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, made.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(ctx, made.ID); !errors.Is(err, mwanachamaforms.ErrFormNotFound) {
		t.Fatalf("run-a Get after Delete = %v, want ErrNotFound", err)
	}
	if titles(t, b)["Shared"] {
		t.Fatal("run-b gained a form from run-a")
	}
}

func TestRemovingARunRemovesItsRowsAndKeepsTheOthers(t *testing.T) {
	mgr, db := newTestManagerDB(t)
	s := testSpec(t, "test")
	ctx := context.Background()
	a, b := forRun(t, mgr, "run-a"), forRun(t, mgr, "run-b")
	composeAimedForm(t, a, mwanachamaforms.Form{Title: "A"})
	composeAimedForm(t, b, mwanachamaforms.Form{Title: "B"})

	removed, err := specstore.RemoveRun(ctx, db, s, "run-a")
	if err != nil || removed == 0 {
		t.Fatalf("removed %d rows, %v", removed, err)
	}
	if rows, _ := specstore.RowsOfRun(ctx, db, s, "run-a"); len(rows) != 0 {
		t.Fatalf("run-a still holds %+v", rows)
	}
	if !titles(t, b)["B"] {
		t.Fatal("run-b lost its form")
	}
}

func TestARunWithoutAnIDIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.ForRun(""); !errors.Is(err, specstore.ErrRunIDEmpty) {
		t.Fatalf("ForRun(\"\") = %v, want ErrRunIDEmpty", err)
	}
}

func TestTwoRunsCanPublishTheSameLinkKeyAndNeitherLeaksIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a, b := forRun(t, mgr, "run-a"), forRun(t, mgr, "run-b")
	fa := newOpenPublicForm(t, a, "census-key")
	fb := newOpenPublicForm(t, b, "census-key")

	_, got, found, err := a.ResolveLinkKey(ctx, "census-key")
	if err != nil || !found || got.ID != fa.ID {
		t.Fatalf("run-a resolves the key to %+v found=%v, %v; want its own form", got, found, err)
	}
	_, got, found, err = b.ResolveLinkKey(ctx, "census-key")
	if err != nil || !found || got.ID != fb.ID {
		t.Fatalf("run-b resolves the key to %+v found=%v, %v; want its own form", got, found, err)
	}
	if _, _, found, _ := mgr.ResolveLinkKey(ctx, "census-key"); found {
		t.Fatal("the live copy resolves a run's link key")
	}
}
