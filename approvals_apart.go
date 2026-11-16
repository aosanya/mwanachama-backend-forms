package mwanachamaforms

import (
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type AgencyApprovals interface {
	TableFor(agency string) string
	Open(db *gorm.DB, agency string) error
}

type ProvisionOption func(*provisionOptions)

type provisionOptions struct {
	agencyApprovals AgencyApprovals
}

func WithAgencyApprovals(store AgencyApprovals) ProvisionOption {
	return func(o *provisionOptions) { o.agencyApprovals = store }
}

const moveBatch = 200

var agencyApprovalColumns = []string{"kind", "summary"}

func holdsAgencyApprovals(db *gorm.DB, table string) (bool, error) {
	for _, column := range agencyApprovalColumns {
		held, err := hasColumn(db, table, column)
		if err != nil || !held {
			return false, err
		}
	}
	return true, nil
}

func giveBackAgencyApprovals(db *gorm.DB, s *spec.Spec, store AgencyApprovals) error {
	if s.MountName() != spec.DefaultMount {
		return nil
	}
	o, ok := s.ByRole(roleApproval)
	if !ok {
		return nil
	}
	swallowedInto := s.TableFor(o)
	if !db.Migrator().HasTable(swallowedInto) {
		return nil
	}
	swallowed, err := holdsAgencyApprovals(db, swallowedInto)
	if err != nil || !swallowed {
		return err
	}
	if store == nil {
		return fmt.Errorf("forms: %s holds agency approvals and no agency approvals store was passed to Provision", swallowedInto)
	}
	if err := store.Open(db, s.Instance); err != nil {
		return fmt.Errorf("forms: open the agency approvals store: %w", err)
	}
	return moveAgencyApprovals(db, swallowedInto, store.TableFor(s.Instance))
}

func moveAgencyApprovals(db *gorm.DB, from, to string) error {
	columns, err := sharedColumns(db, from, to)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		ids, err := idsAbsentFrom(tx, from, to)
		if err != nil {
			return err
		}
		list := strings.Join(columns, ", ")
		for start := 0; start < len(ids); start += moveBatch {
			batch := ids[start:min(start+moveBatch, len(ids))]
			copyStmt := fmt.Sprintf("insert into %s (%s) select %s from %s where id in ?", to, list, list, from)
			if err := tx.Exec(copyStmt, batch).Error; err != nil {
				return fmt.Errorf("forms: give approvals back to %s: %w", to, err)
			}
			if err := tx.Exec(fmt.Sprintf("delete from %s where id in ?", from), batch).Error; err != nil {
				return fmt.Errorf("forms: clear the approvals given back from %s: %w", from, err)
			}
		}
		return nil
	})
}

func idsAbsentFrom(db *gorm.DB, from, to string) ([]string, error) {
	query := fmt.Sprintf("select id from %s where kind is not null and kind <> '' and id not in (select id from %s) order by id", from, to)
	var ids []string
	if err := db.Raw(query).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("forms: read the approvals held in %s: %w", from, err)
	}
	return ids, nil
}

func sharedColumns(db *gorm.DB, from, to string) ([]string, error) {
	held, err := columnNames(db, from)
	if err != nil {
		return nil, err
	}
	wanted, err := columnNames(db, to)
	if err != nil {
		return nil, err
	}
	var shared []string
	for name := range wanted {
		if held[name] {
			shared = append(shared, name)
		}
	}
	sort.Strings(shared)
	return shared, nil
}

func columnNames(db *gorm.DB, table string) (map[string]bool, error) {
	types, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		return nil, fmt.Errorf("forms: read columns of %s: %w", table, err)
	}
	names := make(map[string]bool, len(types))
	for _, t := range types {
		names[t.Name()] = true
	}
	return names, nil
}
