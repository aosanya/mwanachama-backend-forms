package mwanachamaforms

import (
	"context"
	"fmt"
	"slices"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type Declarable func(ctx context.Context, formID string) ([]string, error)

func (m *formManager) UseDeclarable(d Declarable) { m.declarable = d }

func (m *formManager) checkDeclarable(ctx context.Context, formID, chapterID string) error {
	if m.declarable == nil || chapterID == "" {
		return nil
	}
	offered, err := m.declarable(ctx, formID)
	if err != nil {
		return fmt.Errorf("DeclareChapter: %w", err)
	}
	if !slices.Contains(offered, chapterID) {
		return fmt.Errorf("%w: group %q is not one this form offers", ErrInvalidReference, chapterID)
	}
	return nil
}

func (m *formManager) CountDeclarations(ctx context.Context, formID string) ([]models.DeclarationCount, error) {
	held, err := listOf[models.Declaration](m, m.q(ctx, roleDeclaration).Where("form_id = ?", formID), roleDeclaration)
	if err != nil {
		return nil, fmt.Errorf("CountDeclarations: %w", err)
	}
	counts := map[string]int{}
	for _, d := range held {
		counts[d.DeclaredChapterID]++
	}
	out := make([]models.DeclarationCount, 0, len(counts))
	for chapterID, n := range counts {
		out = append(out, models.DeclarationCount{ChapterID: chapterID, Count: n})
	}
	slices.SortFunc(out, func(a, b models.DeclarationCount) int {
		switch {
		case a.ChapterID < b.ChapterID:
			return -1
		case a.ChapterID > b.ChapterID:
			return 1
		}
		return 0
	})
	return out, nil
}

func (m *formManager) ListDeclarations(ctx context.Context, formID string) ([]models.Declaration, error) {
	out, err := listOf[models.Declaration](m, m.q(ctx, roleDeclaration).Where("form_id = ?", formID).Order("created_at"), roleDeclaration)
	if err != nil {
		return nil, fmt.Errorf("ListDeclarations: %w", err)
	}
	return out, nil
}
