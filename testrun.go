package mwanachamaforms

func (m *formManager) ForRun(runID string) (FormManager, error) {
	scoped, err := m.st.ForRun(runID)
	if err != nil {
		return nil, err
	}
	run := &formManager{
		db:               m.db,
		st:               scoped,
		groups:           m.groups,
		names:            m.names,
		standingAudience: m.standingAudience,
		members:          m.members,
		declarable:       m.declarable,
		defaultMaxFiles:  m.defaultMaxFiles,
		formTypes:        m.formTypes,
	}
	run.approvalRequired.Store(m.approvalRequired.Load())
	return run, nil
}
