package mwanachamaforms_test

import (
	"context"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestRegister_FiltersAndCounts(t *testing.T) {
	um := newTestManager(t)
	newDraftForm(t, um)
	f2 := newDraftForm(t, um)
	if _, err := um.Submit(context.Background(), f2.ID, "author"); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	page, err := um.Register(context.Background(), mwanachamaforms.RegisterQuery{ChapterID: "chapter-1"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if page.Total != 2 || page.ByStatus[mwanachamaforms.StatusDraft] != 1 || page.ByStatus[mwanachamaforms.StatusSubmitted] != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestRegister_SearchByTitle(t *testing.T) {
	um := newTestManager(t)
	newDraftForm(t, um) // "Chapter Health Check"

	page, err := um.Register(context.Background(), mwanachamaforms.RegisterQuery{Search: "health"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}

	page, err = um.Register(context.Background(), mwanachamaforms.RegisterQuery{Search: "nonexistent"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestRegister_Pagination(t *testing.T) {
	um := newTestManager(t)
	for i := 0; i < 3; i++ {
		newDraftForm(t, um)
	}
	limit := 2
	page, err := um.Register(context.Background(), mwanachamaforms.RegisterQuery{Limit: &limit})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if page.Total != 3 || len(page.Rows) != 2 {
		t.Fatalf("unexpected page: total=%d rows=%d", page.Total, len(page.Rows))
	}
}
