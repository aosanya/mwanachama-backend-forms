package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

type reportWorld struct {
	fm      mwanachamaforms.FormManager
	h       http.Handler
	formID  string
	readers []readerCall
}

type readerCall struct{ path, commit string }

func newReportWorld(t *testing.T) *reportWorld {
	t.Helper()
	w := &reportWorld{fm: newTestManager(t)}
	w.fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}, "bob": {"ward-2"}}))
	w.fm.UseReportText(func(_ context.Context, gitPath, gitCommit string) (string, error) {
		w.readers = append(w.readers, readerCall{gitPath, gitCommit})
		return "Annual report, " + gitCommit, nil
	})
	f := publish(t, w.fm, composeForm(t, w.fm, "Annual report", "council", []string{"ward-1", "ward-2"}, 1))
	if _, err := w.fm.Close(context.Background(), f.ID, "clerk"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	w.formID = f.ID
	w.h = mountAsCaller(t, w.fm)
	return w
}

func (w *reportWorld) publishTo(t *testing.T, commit string, groups ...string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"git_path": "reports/annual.md", "git_commit": commit, "chapter_ids": groups})
	if rec := callAs(t, w.h, http.MethodPost, "/forms/"+w.formID+"/results", "clerk", string(body)); rec.Code != http.StatusCreated {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func (w *reportWorld) listAs(t *testing.T, caller string) []mwanachamaforms.PublishedReport {
	t.Helper()
	rec := callAs(t, w.h, http.MethodGet, "/forms/reports/mine", caller, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s lists: status = %d, body = %s", caller, rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) == "null" {
		t.Fatalf("%s lists null, not an empty list", caller)
	}
	var out []mwanachamaforms.PublishedReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	return out
}

func (w *reportWorld) textAs(t *testing.T, caller, formID string) *httptest.ResponseRecorder {
	t.Helper()
	return callAs(t, w.h, http.MethodGet, "/forms/"+formID+"/results/text", caller, "")
}

func TestAMemberOfTheGroupListsTheReportAndReadsItsText(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")
	w.publishTo(t, "def", "ward-1")

	listed := w.listAs(t, "alice")
	if len(listed) != 1 {
		t.Fatalf("alice lists %+v", listed)
	}
	got := listed[0]
	if got.FormID != w.formID || got.FormTitle != "Annual report" || got.Version != 2 || got.PublishedAt == "" || got.GitPath != "reports/annual.md" {
		t.Fatalf("listed = %+v", got)
	}

	rec := w.textAs(t, "alice", w.formID)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice reads: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var text mwanachamaforms.ReportText
	if err := json.Unmarshal(rec.Body.Bytes(), &text); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	if text.FormID != w.formID || text.Version != 2 || text.Text != "Annual report, def" || text.Truncated {
		t.Fatalf("text = %+v", text)
	}
	if len(w.readers) != 1 || w.readers[0] != (readerCall{"reports/annual.md", "def"}) {
		t.Fatalf("the host was asked for %+v", w.readers)
	}
}

func TestAPersonOutsideTheGroupListsNothingAndIsRefusedTheText(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")

	if got := w.listAs(t, "bob"); len(got) != 0 {
		t.Fatalf("bob lists %+v", got)
	}
	if got := w.listAs(t, "nobody-in-any-group"); len(got) != 0 {
		t.Fatalf("a stranger lists %+v", got)
	}
	outsider := w.textAs(t, "bob", w.formID)
	missing := w.textAs(t, "bob", "no-such-form")
	if outsider.Code != http.StatusNotFound || outsider.Body.String() != missing.Body.String() {
		t.Fatalf("outsider: %d %q, missing form: %d %q", outsider.Code, outsider.Body.String(), missing.Code, missing.Body.String())
	}
	if len(w.readers) != 0 {
		t.Fatalf("the host read a report for a person who may not: %+v", w.readers)
	}
}

func TestAReportWhoseNewestVersionWentElsewhereIsNeitherListedNorReadable(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")
	w.publishTo(t, "def", "ward-2")

	if got := w.listAs(t, "alice"); len(got) != 0 {
		t.Fatalf("alice lists %+v", got)
	}
	if rec := w.textAs(t, "alice", w.formID); rec.Code != http.StatusNotFound {
		t.Fatalf("alice reads: status = %d", rec.Code)
	}
	if got := w.listAs(t, "bob"); len(got) != 1 || got[0].Version != 2 {
		t.Fatalf("bob lists %+v", got)
	}
}

func TestAFormWithoutAReportIsNotListedAndHasNoText(t *testing.T) {
	w := newReportWorld(t)
	if got := w.listAs(t, "alice"); len(got) != 0 {
		t.Fatalf("alice lists %+v", got)
	}
	if rec := w.textAs(t, "alice", w.formID); rec.Code != http.StatusNotFound {
		t.Fatalf("alice reads: status = %d", rec.Code)
	}
}

func TestAReportTextIsCutAtTheBoundOnACharacterBoundaryAndSaysSo(t *testing.T) {
	cases := []struct {
		name     string
		unit     string
		unitSize int
	}{
		{"two-byte characters", "é", 2},
		{"three-byte characters", "€", 3},
		{"four-byte characters", "😀", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newReportWorld(t)
			huge := "a" + strings.Repeat(c.unit, documentedReportTextBound/c.unitSize+1)
			w.fm.UseReportText(func(context.Context, string, string) (string, error) { return huge, nil })
			w.publishTo(t, "abc", "ward-1")

			rec := w.textAs(t, "alice", w.formID)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %.200s", rec.Code, rec.Body.String())
			}
			var text mwanachamaforms.ReportText
			if err := json.Unmarshal(rec.Body.Bytes(), &text); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !text.Truncated || text.Size != len(huge) {
				t.Fatalf("truncated = %v, size = %d, want size %d", text.Truncated, text.Size, len(huge))
			}
			if len(text.Text) > documentedReportTextBound || len(text.Text) <= documentedReportTextBound-c.unitSize {
				t.Fatalf("len = %d, want within %d bytes under %d", len(text.Text), c.unitSize, documentedReportTextBound)
			}
			if strings.ContainsRune(text.Text, '�') || !strings.HasPrefix(huge, text.Text) {
				t.Fatalf("the cut split a character")
			}
		})
	}
}

func TestAHostThatCannotReadTheStoreAnswersBadGatewayNotAnEmptyText(t *testing.T) {
	w := newReportWorld(t)
	w.publishTo(t, "abc", "ward-1")
	w.fm.UseReportText(nil)
	if rec := w.textAs(t, "alice", w.formID); rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestBothReadsAreToolsGatedByTheirOwnActions(t *testing.T) {
	tools, err := routes.BuildGatedTools(newTestManager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"forms_results_list_mine": "forms.results.list_mine", "forms_results_read_text": "forms.results.read_text"}
	for _, tool := range tools {
		if action, ok := want[tool.Name]; ok {
			if tool.Action != action {
				t.Fatalf("%s is gated by %s", tool.Name, tool.Action)
			}
			delete(want, tool.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("tools missing: %v", want)
	}
}
