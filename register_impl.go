package mwanachamaforms

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

// lastMoved is the stamp the register sorts on: the latest stage a form has
// reached, preferred over the earlier ones. A form that never left draft has
// none, and sorts last rather than first.
func lastMoved(f models.Form) (string, bool) {
	for _, t := range []string{f.ClosedAt, f.PublishedAt, f.ApprovedAt, f.SubmittedAt} {
		if t != "" {
			return t, true
		}
	}
	return "", false
}

func (m *formManager) respondentCount(ctx context.Context, formID string) (int, error) {
	var count int64
	err := m.q(ctx, roleAnswer).Where("form_id = ?", formID).Distinct("member_id").Count(&count).Error
	return int(count), err
}

// currentQuestionCount counts the prompts nothing has superseded, which is
// what a reader would be asked, rather than every revision ever written.
func (m *formManager) currentQuestionCount(ctx context.Context, formID string) (int, error) {
	var count int64
	superseded := m.db.Table(m.table(roleQuestion)).
		Select("supersedes_question_id").Where("supersedes_question_id <> ''")
	err := m.q(ctx, roleQuestion).
		Where("form_id = ? AND id NOT IN (?)", formID, superseded).
		Count(&count).Error
	return int(count), err
}

// Register returns a filtered, sorted, paginated page of forms. The totals
// are computed over everything the filter matched, before the page is cut, so
// a short page still reports how much there is.
func (m *formManager) Register(ctx context.Context, q models.RegisterQuery) (models.RegisterPage, error) {
	query := m.q(ctx, roleForm)
	if q.ChapterID != "" {
		query = query.Where("originator_chapter_id = ?", q.ChapterID)
	}
	if q.Status != "" {
		query = query.Where("status = ?", string(q.Status))
	}
	forms, err := listOf[models.Form](m, query, roleForm)
	if err != nil {
		return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
	}

	needle := strings.ToLower(strings.TrimSpace(q.Search))
	matched := make([]models.Form, 0, len(forms))
	for _, f := range forms {
		if needle != "" && !strings.Contains(strings.ToLower(f.Title), needle) {
			continue
		}
		matched = append(matched, f)
	}

	sort.Slice(matched, func(i, j int) bool {
		ti, oki := lastMoved(matched[i])
		tj, okj := lastMoved(matched[j])
		switch {
		case oki && okj && ti != tj:
			return ti > tj
		case oki != okj:
			return oki
		default:
			return matched[i].ID > matched[j].ID
		}
	})

	page := models.RegisterPage{ByStatus: map[models.Status]int{}}
	for _, f := range matched {
		page.Total++
		page.ByStatus[f.Status]++
		respondents, err := m.respondentCount(ctx, f.ID)
		if err != nil {
			return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
		}
		page.Respondents += respondents
	}

	page.Rows, err = m.registerRows(ctx, window(matched, q))
	if err != nil {
		return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
	}
	return page, nil
}

func window(matched []models.Form, q models.RegisterQuery) []models.Form {
	from := q.Offset
	if from < 0 {
		from = 0
	}
	if from > len(matched) {
		from = len(matched)
	}
	out := matched[from:]
	if q.Limit == nil {
		return out
	}
	n := *q.Limit
	if n < 0 {
		n = 0
	}
	if n < len(out) {
		out = out[:n]
	}
	return out
}

func (m *formManager) registerRows(ctx context.Context, forms []models.Form) ([]models.RegisterRow, error) {
	out := make([]models.RegisterRow, 0, len(forms))
	for _, f := range forms {
		questions, err := m.currentQuestionCount(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		respondents, err := m.respondentCount(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, models.RegisterRow{Form: f, Questions: questions, Respondents: respondents})
	}
	return out, nil
}
