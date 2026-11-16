package mwanachamaforms

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type RegistrationsBackfilled struct {
	Respondents   int64
	Registrations int64
}

func BackfillRegistrations(db *gorm.DB, s *spec.Spec) (RegistrationsBackfilled, error) {
	registrations := tableFor(s, roleRegistration)
	declarations := tableFor(s, roleDeclaration)
	respondents := tableFor(s, roleRespondent)
	stmt := fmt.Sprintf(`insert into %[1]s (id, respondent_id, form_id, registered_at)
select 'backfill:' || d.respondent_id || ':' || d.form_id, d.respondent_id, d.form_id, coalesce(nullif(d.created_at, ''), r.first_seen_at)
from %[2]s d
join %[3]s r on r.id = d.respondent_id
where not exists (select 1 from %[1]s g where g.respondent_id = d.respondent_id and g.form_id = d.form_id)
on conflict do nothing
returning respondent_id`, registrations, declarations, respondents)
	var registered []string
	if err := db.Raw(stmt).Scan(&registered).Error; err != nil {
		return RegistrationsBackfilled{}, fmt.Errorf("forms: backfill registrations: %w", err)
	}
	distinct := make(map[string]bool, len(registered))
	for _, id := range registered {
		distinct[id] = true
	}
	return RegistrationsBackfilled{Respondents: int64(len(distinct)), Registrations: int64(len(registered))}, nil
}

func backfillRegistrationsOnce(db *gorm.DB, s *spec.Spec) error {
	done, err := BackfillRegistrations(db, s)
	if err != nil {
		return err
	}
	if done.Registrations > 0 {
		slog.Info("registrations backfilled for respondents who declared before registration was checked",
			"table", tableFor(s, roleRegistration), "respondents", done.Respondents, "registrations", done.Registrations)
	}
	return nil
}
