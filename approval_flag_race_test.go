package mwanachamaforms_test

import (
	"context"
	"sync"
	"testing"
)

func TestApprovalRequiredCanBeSetWhilePublishesReadIt(t *testing.T) {
	um := newTestManager(t)
	const rounds = 20
	drafts := make([]string, rounds)
	for i := range drafts {
		drafts[i] = newAuthoredDraft(t, um, "author").ID
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			um.UseApprovalRequired(i%2 == 0)
		}
	}()
	go func() {
		defer wg.Done()
		for _, id := range drafts {
			_, _, _ = um.Publish(context.Background(), id, "", "author", "")
		}
	}()
	wg.Wait()
	um.UseApprovalRequired(false)
	f := newAuthoredDraft(t, um, "author")
	if _, _, err := um.Publish(context.Background(), f.ID, "", "author", ""); err != nil {
		t.Fatalf("publish after the flag settled off: %v", err)
	}
}
