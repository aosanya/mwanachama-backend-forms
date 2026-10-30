package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/identity"
)

const identityModule = "forms"

var ErrNoStableIdentity = errors.New("forms: this object has no stable identity")

var identityObjects = []string{roleForm, roleQuestion}

func hasStableIdentity(object string) bool {
	for _, o := range identityObjects {
		if o == object {
			return true
		}
	}
	return false
}

func (m *formManager) questionRows(ctx context.Context, column, value string) ([]map[string]any, error) {
	var rows []map[string]any
	err := m.q(ctx, roleQuestion).Select("id", "supersedes_question_id").Where(column+" = ?", value).Limit(1).Find(&rows).Error
	return rows, err
}

func (m *formManager) rowIDExists(ctx context.Context, role, id string) (bool, error) {
	var rows []map[string]any
	if err := m.q(ctx, role).Select("id").Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (m *formManager) questionChainRoot(ctx context.Context, id string) (string, bool, error) {
	seen := map[string]bool{}
	for current := id; ; {
		if seen[current] {
			return "", false, fmt.Errorf("question %s: its versions loop", id)
		}
		seen[current] = true
		rows, err := m.questionRows(ctx, "id", current)
		if err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}
		previous, _ := rows[0]["supersedes_question_id"].(string)
		if previous == "" {
			return current, true, nil
		}
		current = previous
	}
}

func (m *formManager) currentQuestionVersion(ctx context.Context, root string) (string, error) {
	seen := map[string]bool{}
	for current := root; ; {
		if seen[current] {
			return "", fmt.Errorf("question %s: its versions loop", root)
		}
		seen[current] = true
		rows, err := m.questionRows(ctx, "supersedes_question_id", current)
		if err != nil {
			return "", err
		}
		if len(rows) == 0 {
			return current, nil
		}
		current, _ = rows[0]["id"].(string)
	}
}

func (m *formManager) StableIdentity(ctx context.Context, object, id string) (string, error) {
	if !hasStableIdentity(object) {
		return "", fmt.Errorf("%s: %w", object, ErrNoStableIdentity)
	}
	stable := id
	var found bool
	var err error
	switch object {
	case roleQuestion:
		stable, found, err = m.questionChainRoot(ctx, id)
	default:
		found, err = m.rowIDExists(ctx, object, id)
	}
	if err != nil {
		return "", fmt.Errorf("StableIdentity: %w", err)
	}
	if !found {
		return "", fmt.Errorf("%s %s: %w", object, id, identity.ErrNotFound)
	}
	raw, err := identity.New(identityModule, m.st.Spec().Instance, stable)
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", object, id, err)
	}
	return raw.String(), nil
}

func (m *formManager) Resolve(ctx context.Context, raw string) (string, string, error) {
	r, err := identity.Parse(raw)
	if err != nil {
		return "", "", err
	}
	if r.Module != identityModule || r.Instance != m.st.Spec().Instance {
		return "", "", fmt.Errorf("%s: %w", raw, identity.ErrNotOwned)
	}
	found, err := m.rowIDExists(ctx, roleForm, r.Identity)
	if err != nil {
		return "", "", fmt.Errorf("Resolve: %w", err)
	}
	if found {
		return roleForm, r.Identity, nil
	}
	root, isRoot, err := m.questionChainRoot(ctx, r.Identity)
	if err != nil {
		return "", "", fmt.Errorf("Resolve: %w", err)
	}
	if !isRoot || root != r.Identity {
		return "", "", fmt.Errorf("%s: %w", raw, identity.ErrNotFound)
	}
	current, err := m.currentQuestionVersion(ctx, root)
	if err != nil {
		return "", "", fmt.Errorf("Resolve: %w", err)
	}
	return roleQuestion, current, nil
}
