package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"
	"net/http"
)

const personSendsAction = "forms.submission.list_of_person"

func refuseUnnamedPerson(routes []Route, authorize dispatch.Authorizer) []Route {
	for i := range routes {
		if routes[i].Action != personSendsAction {
			continue
		}
		next := routes[i].Handler
		routes[i].Handler = func(w http.ResponseWriter, r *http.Request) {
			if authorize != nil && authorize(r.Context(), personSendsAction) != nil {
				next(w, r)
				return
			}
			if r.URL.Query().Get("member_id") == "" {
				httpwire.WriteErr(w, http.StatusBadRequest, "member_id: name the person whose sends to list")
				return
			}
			next(w, r)
		}
	}
	return routes
}
