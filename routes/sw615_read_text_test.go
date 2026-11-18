package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

const documentedReportTextBound = 200000

func (w *reportWorld) publishReturning(t *testing.T, commit string, groups ...string) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"git_path": "reports/annual.md", "git_commit": commit, "chapter_ids": groups})
	rec := callAs(t, w.h, http.MethodPost, "/forms/"+w.formID+"/results", "clerk", string(body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal publish: %v (%s)", err, rec.Body.String())
	}
	return out
}

func (w *reportWorld) readTextAs(t *testing.T, caller string) mwanachamaforms.ReportText {
	t.Helper()
	rec := w.textAs(t, caller, w.formID)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s reads: status = %d, body = %.200s", caller, rec.Code, rec.Body.String())
	}
	var text mwanachamaforms.ReportText
	if err := json.Unmarshal(rec.Body.Bytes(), &text); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return text
}

func TestTheTextCarriesTheTitleVersionDateAndPathThatWerePublished(t *testing.T) {
	w := newReportWorld(t)
	w.publishReturning(t, "abc", "ward-1")
	second := w.publishReturning(t, "def", "ward-1")

	text := w.readTextAs(t, "alice")
	if text.FormID != w.formID || text.FormTitle != "Annual report" {
		t.Fatalf("form = %q %q", text.FormID, text.FormTitle)
	}
	if text.Version != 2 || float64(text.Version) != second["version"] {
		t.Fatalf("version = %d, published %v", text.Version, second["version"])
	}
	if text.PublishedAt == "" || text.PublishedAt != second["published_at"] {
		t.Fatalf("published_at = %q, published %v", text.PublishedAt, second["published_at"])
	}
	if text.GitPath != "reports/annual.md" || text.GitPath != second["git_path"] {
		t.Fatalf("git_path = %q, published %v", text.GitPath, second["git_path"])
	}
	if text.Text != "Annual report, def" || text.Size != len(text.Text) || text.Truncated {
		t.Fatalf("text = %q, size = %d, truncated = %v", text.Text, text.Size, text.Truncated)
	}
}

func TestTheSizeIsTheOriginalByteCountNotTheCharacterCount(t *testing.T) {
	w := newReportWorld(t)
	whole := "naïve café €5"
	w.fm.UseReportText(func(context.Context, string, string) (string, error) { return whole, nil })
	w.publishTo(t, "abc", "ward-1")

	text := w.readTextAs(t, "alice")
	if text.Text != whole || text.Size != len(whole) || text.Truncated {
		t.Fatalf("text = %q, size = %d (want %d), truncated = %v", text.Text, text.Size, len(whole), text.Truncated)
	}
}

func TestAFileOfExactlyTheBoundIsNotCutAndOneByteMoreIs(t *testing.T) {
	bound := documentedReportTextBound
	cases := []struct {
		name      string
		size      int
		truncated bool
	}{
		{"one byte under", bound - 1, false},
		{"exactly the bound", bound, false},
		{"one byte over", bound + 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newReportWorld(t)
			whole := strings.Repeat("x", c.size)
			w.fm.UseReportText(func(context.Context, string, string) (string, error) { return whole, nil })
			w.publishTo(t, "abc", "ward-1")

			text := w.readTextAs(t, "alice")
			if text.Size != c.size || text.Truncated != c.truncated {
				t.Fatalf("size = %d, truncated = %v, want %d, %v", text.Size, text.Truncated, c.size, c.truncated)
			}
			wantLen := c.size
			if c.truncated {
				wantLen = bound
			}
			if len(text.Text) != wantLen {
				t.Fatalf("len(text) = %d, want %d", len(text.Text), wantLen)
			}
		})
	}
}
