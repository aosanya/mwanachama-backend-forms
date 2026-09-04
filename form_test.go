package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func futureRFC3339(d time.Duration) string {
	return time.Now().Add(d).UTC().Format(time.RFC3339Nano)
}

func newDraftForm(t *testing.T, um mwanachamaforms.FormManager) models.Form {
	t.Helper()
	f, err := um.Create(context.Background(), models.Form{
		Title:               "Chapter Health Check",
		OriginatorChapterID: "chapter-1",
		ClosesAt:            futureRFC3339(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return f
}

func TestCreate(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if f.ID == "" || f.Status != models.StatusDraft || f.Audience != models.AudienceMember || f.CollectionMode != models.CollectionSelf {
		t.Fatalf("unexpected form: %+v", f)
	}
}

func TestCreate_MissingTitle(t *testing.T) {
	um := newTestManager(t)
	_, err := um.Create(context.Background(), models.Form{ClosesAt: futureRFC3339(time.Hour)})
	if !errors.Is(err, mwanachamaforms.ErrMissingTitle) {
		t.Fatalf("err = %v, want ErrMissingTitle", err)
	}
}

func TestCreate_MissingClosesAt(t *testing.T) {
	um := newTestManager(t)
	_, err := um.Create(context.Background(), models.Form{Title: "x"})
	if !errors.Is(err, mwanachamaforms.ErrMissingClosesAt) {
		t.Fatalf("err = %v, want ErrMissingClosesAt", err)
	}
}

func TestCreate_ClosesInPast(t *testing.T) {
	um := newTestManager(t)
	_, err := um.Create(context.Background(), models.Form{Title: "x", ClosesAt: futureRFC3339(-time.Hour)})
	if !errors.Is(err, mwanachamaforms.ErrClosesInPast) {
		t.Fatalf("err = %v, want ErrClosesInPast", err)
	}
}

func TestCreate_WindowInvalid(t *testing.T) {
	um := newTestManager(t)
	_, err := um.Create(context.Background(), models.Form{
		Title: "x", ClosesAt: futureRFC3339(time.Hour), OpensAt: futureRFC3339(2 * time.Hour),
	})
	if !errors.Is(err, mwanachamaforms.ErrWindowInvalid) {
		t.Fatalf("err = %v, want ErrWindowInvalid", err)
	}
}

func TestCreate_AudienceConflict(t *testing.T) {
	um := newTestManager(t)
	_, err := um.Create(context.Background(), models.Form{
		Title: "x", ClosesAt: futureRFC3339(time.Hour),
		Audience: models.AudienceMember, CollectionMode: models.CollectionInterviewer,
	})
	if !errors.Is(err, mwanachamaforms.ErrAudienceConflict) {
		t.Fatalf("err = %v, want ErrAudienceConflict", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	um := newTestManager(t)
	if _, err := um.Get(context.Background(), "nope"); !errors.Is(err, mwanachamaforms.ErrFormNotFound) {
		t.Fatalf("err = %v, want ErrFormNotFound", err)
	}
}

func TestUpdate(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	f.Title = "Renamed"
	out, err := um.Update(context.Background(), f)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Title != "Renamed" {
		t.Fatalf("unexpected form: %+v", out)
	}
}

func TestUpdate_NotDraft(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	f.Title = "Renamed"
	if _, err := um.Update(context.Background(), f); !errors.Is(err, mwanachamaforms.ErrNotDraft) {
		t.Fatalf("err = %v, want ErrNotDraft", err)
	}
}

func TestListForChapter(t *testing.T) {
	um := newTestManager(t)
	newDraftForm(t, um)
	newDraftForm(t, um)
	out, err := um.ListForChapter(context.Background(), "chapter-1")
	if err != nil {
		t.Fatalf("ListForChapter: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d forms, want 2", len(out))
	}
}

func TestPublish_MemberAudienceCreatesPropagation(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.AddTarget(context.Background(), models.Target{FormID: f.ID, ChapterID: "chapter-2"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}

	out, link, err := um.Publish(context.Background(), f.ID, "", "admin")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if out.Status != models.StatusOpen || link.ID != "" {
		t.Fatalf("unexpected publish result: %+v %+v", out, link)
	}
	rollup, err := um.Rollup(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("Rollup: %v", err)
	}
	if rollup.Targeted != 1 || rollup.Stalled != 1 {
		t.Fatalf("unexpected rollup: %+v", rollup)
	}
}

func TestPublish_PublicAudienceCreatesLink(t *testing.T) {
	um := newTestManager(t)
	f, err := um.Create(context.Background(), models.Form{
		Title: "Public Census", OriginatorChapterID: "chapter-1",
		ClosesAt: futureRFC3339(time.Hour), Audience: models.AudiencePublic,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	out, link, err := um.Publish(context.Background(), f.ID, "abc123", "admin")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if out.Status != models.StatusOpen || link.Key != "abc123" || link.Status != models.LinkActive {
		t.Fatalf("unexpected publish result: %+v %+v", out, link)
	}
}

func TestPublish_AwaitingApproval(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); !errors.Is(err, mwanachamaforms.ErrAwaitingApproval) {
		t.Fatalf("err = %v, want ErrAwaitingApproval", err)
	}
}

func TestClose(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	out, err := um.Close(context.Background(), f.ID, "admin")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if out.Status != models.StatusClosed {
		t.Fatalf("unexpected form: %+v", out)
	}
}

func TestClose_NotOpen(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, err := um.Close(context.Background(), f.ID, "admin"); !errors.Is(err, mwanachamaforms.ErrNotOpen) {
		t.Fatalf("err = %v, want ErrNotOpen", err)
	}
}

func TestDelete(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if err := um.Delete(context.Background(), f.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := um.Get(context.Background(), f.ID); !errors.Is(err, mwanachamaforms.ErrFormNotFound) {
		t.Fatalf("err = %v, want ErrFormNotFound", err)
	}
}

func TestDelete_Published(t *testing.T) {
	um := newTestManager(t)
	f := newDraftForm(t, um)
	if _, _, err := um.Publish(context.Background(), f.ID, "", "admin"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := um.Delete(context.Background(), f.ID); !errors.Is(err, mwanachamaforms.ErrPublished) {
		t.Fatalf("err = %v, want ErrPublished", err)
	}
}
