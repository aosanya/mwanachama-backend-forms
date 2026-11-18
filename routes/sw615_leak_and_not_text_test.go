package routes_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAReaderFailureNeverShowsTheHostsReasonToTheResident(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")
	w.fm.UseReportText(func(context.Context, string, string) (string, error) {
		return "", errors.New("open /srv/git/instance-77/reports.git: permission denied token=SECRET")
	})

	rec := w.textAs(t, "alice", w.formID)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	for _, leaked := range []string{"/srv/git", "instance-77", "permission denied", "SECRET"} {
		if strings.Contains(rec.Body.String(), leaked) {
			t.Fatalf("the response carries %q: %s", leaked, rec.Body.String())
		}
	}
	if !strings.Contains(rec.Body.String(), "cannot be read now") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestAFileThatIsNotTextIsRefusedNotServedAsMangledText(t *testing.T) {
	cases := map[string]string{
		"a zip header":                "PK\x03\x04\x00\xff\xfe\xfa",
		"only continuation bytes":     strings.Repeat("\x80", 300000),
		"valid text then a bad byte":  "Annual report \xc3(",
		"a cut that would land in it": "a" + strings.Repeat("\x80", 200010),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			w := newReportWorld(t)
			w.publishTo(t, "abc", "ward-1")
			w.fm.UseReportText(func(context.Context, string, string) (string, error) { return content, nil })

			rec := w.textAs(t, "alice", w.formID)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, body = %.200s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "\"text\"") {
				t.Fatalf("the refusal still carries text: %.200s", rec.Body.String())
			}
		})
	}
}

func TestAnOutsiderIsRefusedBeforeTheFileIsJudgedNotText(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")
	w.fm.UseReportText(func(context.Context, string, string) (string, error) { return "\xff\xfe", nil })

	rec := w.textAs(t, "bob", w.formID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestAnEmptyFileIsStillText(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")
	w.fm.UseReportText(func(context.Context, string, string) (string, error) { return "", nil })

	rec := w.textAs(t, "alice", w.formID)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "\"truncated\":false") {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
