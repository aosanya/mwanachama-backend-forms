package mwanachamaforms

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

// The roles a caller may name when checking a value against a spec without a
// database — see [Check]. They are the module's own words for the objects a
// domain fills; the unexported constants in store.go are the same strings.
const (
	// RoleForm is the instrument itself.
	RoleForm = roleForm

	// RoleQuestion is one prompt on it.
	RoleQuestion = roleQuestion

	// RoleAnswer is one response to one prompt.
	RoleAnswer = roleAnswer
)

// Check reports whether v satisfies what s declares for the object playing
// role. It needs no database, which is what lets a bulk import validate
// everything it has read before opening a connection.
func Check(s *spec.Spec, role string, v any) error {
	o, ok := s.ByRole(role)
	if !ok {
		return fmt.Errorf("check: this domain fills no object for the role %q", role)
	}
	return check(o, v)
}

// check reports whether v satisfies what o declares: a required field that is
// present, and an enum value that is one of the declared ones.
//
// These are the rules the spec can state, so they are read from it rather
// than written again in Go. What stays in Go is what a spec cannot say — that
// a closing time falls after an opening one, that a member audience is never
// interviewed, that an answer's value fits the shape its own question asks
// for — and each of those lives with the type it is about.
func check(o spec.Object, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("check %s: want a struct, got %T", o.Name, v)
	}

	byColumn := map[string]reflect.Value{}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).PkgPath != "" {
			continue
		}
		byColumn[columnName(rt.Field(i).Name)] = rv.Field(i)
	}

	for _, f := range o.Fields {
		fv, ok := byColumn[f.Name]
		if !ok || fv.Kind() != reflect.String {
			continue
		}
		s := fv.String()

		if f.Required && strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s: %s is required", o.Name, f.Name)
		}
		if f.Type == spec.TypeEnum && !declares(f.Values, s) {
			return fmt.Errorf("%s: %s is %q, which is not one of %s",
				o.Name, f.Name, s, strings.Join(f.Values, ", "))
		}
	}
	return nil
}

func declares(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

func (m *formManager) checks(role string, v any) error {
	if err := check(m.st.Object(role), v); err != nil {
		return fmt.Errorf("%w: %v", invalidFor(role), err)
	}
	return nil
}

func invalidFor(role string) error {
	switch role {
	case roleQuestion:
		return ErrInvalidQuestion
	case roleAnswer:
		return ErrInvalidAnswer
	default:
		return ErrInvalidForm
	}
}
