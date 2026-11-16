package mwanachamaforms

import (
	"context"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func checkLandsIn(l models.LandsIn) error {
	if !l.Declared() {
		return nil
	}
	if err := l.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidForm, err)
	}
	if _, _, err := l.Target(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidForm, err)
	}
	return nil
}

func (m *formManager) writerFor(l models.LandsIn) (RecordWriter, error) {
	module, _, err := l.Target()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidForm, err)
	}
	if m.recordWriters != nil {
		if w, found := m.recordWriters(module); found {
			return w, nil
		}
	}
	return nil, fmt.Errorf("%w %q", ErrNoRecordWriter, module)
}

func (m *formManager) refuseUnlandable(ctx context.Context, f models.Form) error {
	if !f.LandsIn.Declared() {
		return nil
	}
	if err := checkLandsIn(f.LandsIn); err != nil {
		return err
	}
	questions, err := m.currentQuestions(ctx, f.ID)
	if err != nil {
		return fmt.Errorf("Publish: %w", err)
	}
	if missing := f.LandsIn.UnmappedQuestions(questions); len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrUnmappedQuestions, strings.Join(missing, ", "))
	}
	writer, err := m.writerFor(f.LandsIn)
	if err != nil {
		return err
	}
	if err := checkWriterFields(writer, f.LandsIn.Record, f.LandsIn.Fields()); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidForm, err)
	}
	return nil
}

func (m *formManager) land(ctx context.Context, f models.Form, sent sentAnswers) error {
	if !f.LandsIn.Declared() {
		return nil
	}
	writer, err := m.writerFor(f.LandsIn)
	if err != nil {
		return err
	}
	record, err := f.LandsIn.Land(sent.sources(), sent.context(f.ID))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLandingFailed, err)
	}
	if err := writer.Write(ctx, record); err != nil {
		return fmt.Errorf("%w: %v", ErrLandingFailed, err)
	}
	return nil
}

type sentAnswers struct {
	answers   []models.Answer
	questions map[string]models.Question
	options   map[string][]models.QuestionOption
	memberID  string
	sendID    string
	period    string
}

func (s sentAnswers) sources() []models.LandingSource {
	out := make([]models.LandingSource, 0, len(s.answers))
	for _, a := range s.answers {
		out = append(out, models.LandingSource{Question: s.questions[a.QuestionID], Options: s.options[a.QuestionID], Answer: a})
	}
	return out
}

func (s sentAnswers) context(formID string) map[string]string {
	return map[string]string{"form": formID, "member": s.memberID, "send": s.sendID, "period": s.period}
}
