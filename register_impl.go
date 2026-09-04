// register_impl.go — the search/list page. Ported from
// mwanachama-backend-api-gateway's
// internal/store/memory/survey_store_register.go.
package mwanachamaforms

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aosanya/mwanachama-backend-forms/gormstore"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// lastMoved is the timestamp Register sorts by: the first of
// ClosedAt/PublishedAt/ApprovedAt/SubmittedAt that is set, in that priority
// order. A form with none of these (never moved past draft) sorts last.
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
	err := m.db.WithContext(ctx).Table(m.tables.Answers).Where("form_id = ?", formID).Distinct("member_id").Count(&count).Error
	return int(count), err
}

// currentQuestionCount counts a form's live (not superseded) questions.
func (m *formManager) currentQuestionCount(ctx context.Context, formID string) (int, error) {
	var count int64
	sub := m.db.Table(m.tables.Questions).Select("supersedes_question_id").Where("supersedes_question_id <> ''")
	err := m.db.WithContext(ctx).Table(m.tables.Questions).
		Where("form_id = ? AND id NOT IN (?)", formID, sub).
		Count(&count).Error
	return int(count), err
}

// Register returns a filtered, sorted, paginated page of Forms. Sort is by
// lastMoved descending, ties (or two never-moved forms) broken by id
// descending. Total/ByStatus/Respondents are computed over the whole
// filtered set, before Limit/Offset apply.
func (m *formManager) Register(ctx context.Context, q models.RegisterQuery) (models.RegisterPage, error) {
	query := m.db.WithContext(ctx).Table(m.tables.Forms)
	if q.ChapterID != "" {
		query = query.Where("originator_chapter_id = ?", q.ChapterID)
	}
	if q.Status != "" {
		query = query.Where("status = ?", string(q.Status))
	}
	var rows []gormstore.FormRow
	if err := query.Find(&rows).Error; err != nil {
		return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
	}

	needle := strings.ToLower(strings.TrimSpace(q.Search))
	matched := make([]models.Form, 0, len(rows))
	for _, r := range rows {
		f := gormstore.FormFromRow(r)
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
		rc, err := m.respondentCount(ctx, f.ID)
		if err != nil {
			return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
		}
		page.Respondents += rc
	}

	from := q.Offset
	if from < 0 {
		from = 0
	}
	if from > len(matched) {
		from = len(matched)
	}
	window := matched[from:]
	if q.Limit != nil {
		n := *q.Limit
		if n < 0 {
			n = 0
		}
		if n < len(window) {
			window = window[:n]
		}
	}

	page.Rows = make([]models.RegisterRow, 0, len(window))
	for _, f := range window {
		qc, err := m.currentQuestionCount(ctx, f.ID)
		if err != nil {
			return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
		}
		rc, err := m.respondentCount(ctx, f.ID)
		if err != nil {
			return models.RegisterPage{}, fmt.Errorf("Register: %w", err)
		}
		page.Rows = append(page.Rows, models.RegisterRow{Form: f, Questions: qc, Respondents: rc})
	}
	return page, nil
}
