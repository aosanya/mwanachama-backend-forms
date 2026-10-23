package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type Members func(ctx context.Context, groupID string, includesDescendants bool) ([]string, error)

func (m *formManager) UseMembers(mb Members) { m.members = mb }

func (m *formManager) ListMissed(ctx context.Context, formID, periodKey string) ([]models.MissedMember, error) {
	f, err := m.Get(ctx, formID)
	if err != nil {
		return nil, err
	}
	if !f.Schedule.Repeats() {
		return nil, ErrNotRepeating
	}
	if err := periodHasClosed(f.Schedule, periodKey, time.Now()); err != nil {
		return nil, err
	}
	if m.members == nil {
		return []models.MissedMember{}, nil
	}
	sent, err := m.sendersIn(ctx, formID, periodKey)
	if err != nil {
		return nil, fmt.Errorf("ListMissed: %w", err)
	}
	targets, err := m.ListTargets(ctx, formID)
	if err != nil {
		return nil, fmt.Errorf("ListMissed: %w", err)
	}
	out := []models.MissedMember{}
	seen := map[string]bool{}
	for _, t := range targets {
		ids, err := m.members(ctx, t.ChapterID, t.IncludesDescendants)
		if err != nil {
			return nil, fmt.Errorf("ListMissed: %w", err)
		}
		for _, id := range ids {
			if sent[id] || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, models.MissedMember{MemberID: id, ChapterID: t.ChapterID})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MemberID < out[j].MemberID })
	return out, nil
}

func periodHasClosed(s models.FormSchedule, periodKey string, now time.Time) error {
	if periodKey == "" {
		return ErrPeriodNotClosed
	}
	current, err := s.PeriodAt(now)
	if errors.Is(err, models.ErrNoPeriod) {
		return nil
	}
	if err != nil {
		return err
	}
	if periodKey >= current.Key {
		return ErrPeriodNotClosed
	}
	return nil
}

func (m *formManager) sendersIn(ctx context.Context, formID, periodKey string) (map[string]bool, error) {
	subs, err := m.ListSubmissions(ctx, formID, "", periodKey)
	if err != nil {
		return nil, err
	}
	sent := map[string]bool{}
	for _, s := range subs {
		sent[s.MemberID] = true
	}
	return sent, nil
}
