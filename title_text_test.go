package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestAFormTitleWithANulByteOrAControlCharacterIsRefused(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	for label, title := range map[string]string{
		"nul":     "Lakossági\u0000kérdőív",
		"bell":    "Lakossági\u0007kérdőív",
		"newline": "Lakossági\nkérdőív",
	} {
		t.Run(label, func(t *testing.T) {
			_, err := m.Create(ctx, models.Form{Title: title, ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1"})
			if !errors.Is(err, forms.ErrInvalidForm) {
				t.Fatalf("Create with %q = %v, want ErrInvalidForm", title, err)
			}
		})
	}
	if _, err := m.Create(ctx, models.Form{Title: "Lakossági kérdőív", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1"}); err != nil {
		t.Fatalf("Create with a plain title = %v", err)
	}
}
