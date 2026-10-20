package mwanachamaforms_test

import (
	"context"
	"testing"
)

func TestRegisteringWithoutAKeyMintsADistinctKeyEachTime(t *testing.T) {
	um := newTestManager(t)
	newOpenPublicForm(t, um, "shared-link")
	ctx := context.Background()

	first, err := um.RegisterRespondent(ctx, "shared-link", "")
	if err != nil {
		t.Fatalf("first key-less registration: %v", err)
	}
	second, err := um.RegisterRespondent(ctx, "shared-link", "")
	if err != nil {
		t.Fatalf("second key-less registration: %v", err)
	}
	if first.PublicKey == "" || second.PublicKey == "" {
		t.Fatalf("a key-less registration returned no key: %q, %q", first.PublicKey, second.PublicKey)
	}
	if first.PublicKey == second.PublicKey || first.ID == second.ID {
		t.Fatalf("two key-less visitors share a respondent: %+v vs %+v", first, second)
	}

	again, err := um.RegisterRespondent(ctx, "shared-link", first.PublicKey)
	if err != nil {
		t.Fatalf("re-registering a minted key: %v", err)
	}
	if again.ID != first.ID || again.PublicKey != first.PublicKey {
		t.Fatalf("a minted key did not come back as it stands: %+v vs %+v", again, first)
	}
}

func TestRegisteringWithAKeyKeepsThatKey(t *testing.T) {
	um := newTestManager(t)
	newOpenPublicForm(t, um, "shared-link")

	r, err := um.RegisterRespondent(context.Background(), "shared-link", "device-held-key")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	if r.PublicKey != "device-held-key" {
		t.Fatalf("a supplied key was replaced: %q", r.PublicKey)
	}
}
