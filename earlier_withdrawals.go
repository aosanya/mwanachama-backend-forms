package mwanachamaforms

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type EarlierWithdrawal struct {
	MemberID    string
	WithdrawnAt string
}

type EarlierWithdrawalsStamped struct {
	Members int64
	Sends   int64
	Answers int64
}

func StampEarlierWithdrawals(db *gorm.DB, s *spec.Spec, withdrawn []EarlierWithdrawal) (EarlierWithdrawalsStamped, error) {
	var total EarlierWithdrawalsStamped
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, w := range withdrawn {
			if w.MemberID == "" || w.WithdrawnAt == "" {
				continue
			}
			sends, err := stampLiveRows(tx, tableFor(s, roleSubmission), w)
			if err != nil {
				return err
			}
			answers, err := stampLiveRows(tx, tableFor(s, roleAnswer), w)
			if err != nil {
				return err
			}
			if sends+answers > 0 {
				total.Members++
			}
			total.Sends += sends
			total.Answers += answers
		}
		return nil
	})
	if err != nil {
		return EarlierWithdrawalsStamped{}, err
	}
	return total, nil
}

func stampLiveRows(tx *gorm.DB, table string, w EarlierWithdrawal) (int64, error) {
	res := tx.Table(table).
		Where("member_id = ? AND withdrawn_at = ''", w.MemberID).
		Update("withdrawn_at", w.WithdrawnAt)
	if res.Error != nil {
		return 0, fmt.Errorf("forms: stamp the earlier withdrawal of %s in %s: %w", w.MemberID, table, res.Error)
	}
	return res.RowsAffected, nil
}
