package mwanachamaforms_test

import (
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

func TestPublicDoor_ACallerChosenKeyIsRefusedAndNoRespondentIsCreated(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := t.Context()
	_, key := publicRound(t, m)
	if _, err := m.RegisterRespondent(ctx, key, "chosen-by-caller"); !errors.Is(err, forms.ErrInvalidReference) {
		t.Fatalf("registering with a key nobody was issued = %v, want ErrInvalidReference", err)
	}
	minted, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	if minted.PublicKey == "chosen-by-caller" || len(minted.PublicKey) != 64 {
		t.Fatalf("the server must mint the key, got %q", minted.PublicKey)
	}
}

func TestPublicDoor_TwoRegistrationsWithoutAKeyGetDifferentMintedKeys(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := t.Context()
	_, key := publicRound(t, m)
	first, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.PublicKey == second.PublicKey || first.ID == second.ID {
		t.Fatalf("two devices shared a respondent: %+v %+v", first, second)
	}
}

func TestPublicDoor_AnIssuedKeyStillReclaimsItsRespondent(t *testing.T) {
	m := gatedManager(t, gateFor{})
	ctx := t.Context()
	_, key := publicRound(t, m)
	first, err := m.RegisterRespondent(ctx, key, "")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	again, err := m.RegisterRespondent(ctx, key, first.PublicKey)
	if err != nil || again.ID != first.ID {
		t.Fatalf("reclaiming with the issued key = %+v, %v", again, err)
	}
}
