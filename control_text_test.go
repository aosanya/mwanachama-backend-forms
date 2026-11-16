package mwanachamaforms_test

import (
	"context"
	"errors"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestAnApprovalNoteWithAControlByteIsRefusedAndALegitimateMultiLineNoteIsKept(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, nil)
	ctx := context.Background()

	for _, note := range []string{"no\u0000te", "no\u0007te"} {
		if _, err := m.Refuse(ctx, f.ID, "approver-1", note); !errors.Is(err, forms.ErrInvalidForm) {
			t.Fatalf("Refuse with note %q = %v, want ErrInvalidForm", note, err)
		}
		if _, err := m.Approve(ctx, f.ID, "approver-1", note); !errors.Is(err, forms.ErrInvalidForm) {
			t.Fatalf("Approve with note %q = %v, want ErrInvalidForm", note, err)
		}
	}
	if _, err := m.Refuse(ctx, f.ID, "approver-1", "A költségvetési sor hiányzik.\nKérjük pótolni."); err != nil {
		t.Fatalf("Refuse with a two-line Hungarian note = %v", err)
	}
}

func TestATranslationWithAControlByteIsRefused(t *testing.T) {
	m := gatedManager(t, nil)
	f := submittedWithAmount(t, m, 0, nil)
	ctx := context.Background()
	put := func(text string) error {
		_, err := m.PutTranslation(ctx, models.Translation{
			FormID: f.ID, SubjectRole: models.TranslationOfForm, SubjectID: f.ID, Field: "title", Language: "hu", Text: text,
		})
		return err
	}
	if err := put("Lakossági\u0000kérdőív"); !errors.Is(err, forms.ErrInvalidForm) {
		t.Fatalf("PutTranslation with a NUL byte = %v, want ErrInvalidForm", err)
	}
	if err := put("Lakossági kérdőív"); err != nil {
		t.Fatalf("PutTranslation with plain text = %v", err)
	}
}
