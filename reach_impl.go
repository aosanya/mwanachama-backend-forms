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

func narrowedTo(ctx context.Context) (string, bool) {
	callerID, narrowed := ctx.Value(reachedOnlyKey{}).(string)
	return callerID, narrowed
}

func (m *formManager) hiddenFromCaller(ctx context.Context, f models.Form) (bool, error) {
	callerID, narrowed := narrowedTo(ctx)
	if !narrowed {
		return false, nil
	}
	if !models.VisibleToMembers(f.Status) || m.groups == nil || callerID == "" {
		return true, nil
	}
	groups, err := m.groups(ctx, callerID)
	if err != nil {
		return false, err
	}
	if len(groups) == 0 {
		return true, nil
	}
	reached, err := m.reachedThrough(ctx, groups)
	if err != nil {
		return false, err
	}
	_, reaches := reached[f.ID]
	return !reaches, nil
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
