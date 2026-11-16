package mwanachamaforms

import (
	"context"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

type RecordWriter interface {
	Write(ctx context.Context, record models.LandedRecord) error
}

type FieldChecker interface {
	CheckFields(record string, fields []string) error
}

type RecordWriters func(module string) (RecordWriter, bool)

func (w RecordWriters) Discarding() RecordWriters {
	if w == nil {
		return nil
	}
	return func(module string) (RecordWriter, bool) {
		inner, found := w(module)
		if !found {
			return nil, false
		}
		return discardingWriter{inner: inner}, true
	}
}

type discardingWriter struct{ inner RecordWriter }

func (discardingWriter) Write(context.Context, models.LandedRecord) error { return nil }

func (d discardingWriter) CheckFields(record string, fields []string) error {
	return checkWriterFields(d.inner, record, fields)
}

func checkWriterFields(w RecordWriter, record string, fields []string) error {
	if checker, ok := w.(FieldChecker); ok {
		return checker.CheckFields(record, fields)
	}
	return nil
}

func (m *formManager) UseRecordWriters(w RecordWriters) { m.recordWriters = w }
