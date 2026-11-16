package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type RecordWriter interface {
	Write(ctx context.Context, record models.LandedRecord) error
}

type RecordWriters func(module string) (RecordWriter, bool)

func (w RecordWriters) Discarding() RecordWriters {
	if w == nil {
		return nil
	}
	return func(module string) (RecordWriter, bool) {
		if _, found := w(module); !found {
			return nil, false
		}
		return discardingWriter{}, true
	}
}

type discardingWriter struct{}

func (discardingWriter) Write(context.Context, models.LandedRecord) error { return nil }

func (m *formManager) UseRecordWriters(w RecordWriters) { m.recordWriters = w }
