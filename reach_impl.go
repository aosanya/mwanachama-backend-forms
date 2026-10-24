package mwanachamaforms

import (
	"context"
	"errors"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type reachedOnlyKey struct{}

func OnlyReached(ctx context.Context, callerID string) context.Context {
	return context.WithValue(ctx, reachedOnlyKey{}, callerID)
}

func unnarrowed(ctx context.Context) context.Context {
	return context.WithValue(ctx, reachedOnlyKey{}, nil)
}

func narrowedTo(ctx context.Context) (string, bool) {
	callerID, narrowed := ctx.Value(reachedOnlyKey{}).(string)
	return callerID, narrowed
}

func (m *formManager) hiddenFromCaller(ctx context.Context, f models.Form) (bool, error) {
	callerID, narrowed := narrowedTo(ctx)
	if !narrowed {
		return false, nil
	}
	if !models.VisibleToMembers(f.Status) || callerID == "" {
		return true, nil
	}
	reaches, err := m.reaches(ctx, callerID, f)
	if err != nil || reaches {
		return false, err
	}
	came, err := m.cameByLink(ctx, callerID, f.ID)
	return !came, err
}

func (m *formManager) reaches(ctx context.Context, callerID string, f models.Form) (bool, error) {
	if f.Standing {
		group, err := m.standingGroup(ctx, callerID, f.ID, "")
		return group != "", err
	}
	return m.reachesThroughGroups(ctx, callerID, f.ID)
}

func (m *formManager) reachesThroughGroups(ctx context.Context, callerID, formID string) (bool, error) {
	if m.groups == nil {
		return false, nil
	}
	groups, err := m.groups(ctx, callerID)
	if err != nil || len(groups) == 0 {
		return false, err
	}
	reached, err := m.reachedThrough(ctx, groups)
	if err != nil {
		return false, err
	}
	_, reaches := reached[formID]
	return reaches, nil
}

func (m *formManager) formHiddenFromCaller(ctx context.Context, formID string) (bool, error) {
	if _, narrowed := narrowedTo(ctx); !narrowed {
		return false, nil
	}
	if _, err := m.Get(ctx, formID); err != nil {
		if errors.Is(err, ErrFormNotFound) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (m *formManager) questionHiddenFromCaller(ctx context.Context, questionID string) (bool, error) {
	if _, narrowed := narrowedTo(ctx); !narrowed {
		return false, nil
	}
	q, err := m.findQuestion(ctx, questionID)
	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			return true, nil
		}
		return false, err
	}
	return m.formHiddenFromCaller(ctx, q.FormID)
}
