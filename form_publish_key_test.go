package mwanachamaforms_test

import (
	"testing"
)

func TestPublish_TwoPublicFormsPublishedWithoutAKeyGetDistinctMintedKeys(t *testing.T) {
	m := gatedManager(t, gateFor{})
	_, first := publicRound(t, m)
	_, second := publicRound(t, m)
	if first == "" || second == "" {
		t.Fatalf("a public form published without a key must get a minted one, got %q and %q", first, second)
	}
	if first == second {
		t.Fatalf("two public forms share the link key %q", first)
	}
}
