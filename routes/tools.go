package routes

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

type Tool = dispatch.Tool

func BuildGatedTools(fm forms.FormManager, m Mount) ([]Tool, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	all, err := dispatch.Tools(s, dispatch.Deps{
		Manager: fm, Errors: sentinels, Authorize: m.Authorize, Caller: m.Caller,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Tool, 0, len(all))
	for _, t := range all {
		if slices.Contains(AnonymousActions, t.Action) {
			continue
		}
		if m.ReachedOnly {
			t = reachedOnlyTool(t, m.Caller)
		}
		out = append(out, t)
	}
	return out, nil
}

func reachedOnlyTool(t Tool, caller dispatch.Caller) Tool {
	narrow := func(ctx context.Context) context.Context {
		callerID := ""
		if caller != nil {
			callerID = caller(ctx)
		}
		return forms.OnlyReached(ctx, callerID)
	}
	invoke, handle := t.Invoke, t.Handle
	if invoke != nil {
		t.Invoke = func(ctx context.Context, raw json.RawMessage) (any, error) { return invoke(narrow(ctx), raw) }
	}
	if handle != nil {
		t.Handle = func(ctx context.Context, req dispatch.Request) (any, error) { return handle(narrow(ctx), req) }
	}
	return t
}
