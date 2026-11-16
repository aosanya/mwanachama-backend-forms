package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type windowSpy struct {
	forms.FormManager
	registerCalls int
}

func (s *windowSpy) Register(context.Context, models.RegisterQuery) (models.RegisterPage, error) {
	s.registerCalls++
	return models.RegisterPage{}, nil
}

func registerStatusFor(t *testing.T, spy *windowSpy, rawQuery string) int {
	t.Helper()
	for _, route := range RoutesFor(spy, Mount{}) {
		if route.Action != "forms.form.list" {
			continue
		}
		req := httptest.NewRequest(http.MethodGet, "/forms?"+rawQuery, nil)
		rec := httptest.NewRecorder()
		route.Handler(rec, req)
		return rec.Code
	}
	t.Fatal("no route for forms.form.list")
	return 0
}

func TestTheFormsRegisterRefusesAWindowThatCannotBeAPage(t *testing.T) {
	for _, rawQuery := range []string{"limit=0", "limit=-1", "offset=-1", "limit=1&offset=-3"} {
		spy := &windowSpy{}
		if got := registerStatusFor(t, spy, rawQuery); got != http.StatusBadRequest {
			t.Errorf("GET /forms?%s answered %d, want 400", rawQuery, got)
		}
		if spy.registerCalls != 0 {
			t.Errorf("GET /forms?%s reached the register", rawQuery)
		}
	}
}

func TestTheFormsRegisterAnswersAWindowThatCanBeAPage(t *testing.T) {
	for _, rawQuery := range []string{"", "limit=1", "limit=25&offset=0", "offset=2"} {
		spy := &windowSpy{}
		if got := registerStatusFor(t, spy, rawQuery); got != http.StatusOK {
			t.Errorf("GET /forms?%s answered %d, want 200", rawQuery, got)
		}
	}
}
