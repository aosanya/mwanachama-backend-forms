package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"sort"
)

func (m *formManager) ListSubmissionsOfPerson(ctx context.Context, memberID, recordRowID string) ([]models.Submission, error) {
	if memberID == "" {
		return nil, fmt.Errorf("ListSubmissionsOfPerson: a member id is required: %w", ErrInvalidAnswer)
	}
	sends, err := m.submissionsBy(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("ListSubmissionsOfPerson: %w", err)
	}
	titles := map[string]string{}
	counted := map[string]map[sentAs]int{}
	out := make([]models.Submission, 0, len(sends))
	for _, send := range sends {
		if recordRowID != "" && send.RecordRowID != recordRowID {
			continue
		}
		title, readable, err := m.titleOfForm(ctx, titles, send.FormID)
		if err != nil {
			return nil, fmt.Errorf("ListSubmissionsOfPerson: %w", err)
		}
		if !readable {
			continue
		}
		if _, loaded := counted[send.FormID]; !loaded {
			if counted[send.FormID], err = m.answerCounts(ctx, send.FormID, memberID); err != nil {
				return nil, fmt.Errorf("ListSubmissionsOfPerson: %w", err)
			}
		}
		send.Answers = counted[send.FormID][sentAs{send.MemberID, send.ChapterID, send.PeriodKey, send.RepeatOf}]
		send.Paid = send.PaidAt != ""
		send.FormTitle = title
		out = append(out, send)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SubmittedAt != out[j].SubmittedAt {
			return out[i].SubmittedAt > out[j].SubmittedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
func (m *formManager) titleOfForm(ctx context.Context, known map[string]string, formID string) (string, bool, error) {
	if title, ok := known[formID]; ok {
		return title, true, nil
	}
	form, err := m.Get(ctx, formID)
	if errors.Is(err, ErrFormNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	known[formID] = form.Title
	return form.Title, true, nil
}
