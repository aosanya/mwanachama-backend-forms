package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

func namesOf(byCaller map[string]string) mwanachamaforms.Names {
	return func(_ context.Context, callerID string) (string, error) {
		return byCaller[callerID], nil
	}
}

func publishAs(t *testing.T, h http.Handler, caller, formID, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/forms/"+formID+"/publish", strings.NewReader(body))
	r.Header.Set("X-Test-Caller", caller)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func getFormAs(t *testing.T, h http.Handler, caller, formID string) mwanachamaforms.Form {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/forms/"+formID, nil)
	r.Header.Set("X-Test-Caller", caller)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("get form: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var f mwanachamaforms.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return f
}

func TestPublishRecordsTheCallersNameAndTheAuthorityItIsSentFor(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"resident-1": {"ward-1"}}))
	fm.UseNames(namesOf(map[string]string{
		"treasurer-1": "Wanjiku Kamau",
		"impostor":    "Somebody Else",
	}))
	h := mountAsCaller(t, fm)
	f := composeForm(t, fm, "Borehole contribution", "council", []string{"ward-1"}, 1)

	rec := publishAs(t, h, "treasurer-1", f.ID, `{
		"published_by": "impostor",
		"publisher_name": "Forged Name",
		"publishing_authority": "  Treasurer, Kilimani Ward  "
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Form mwanachamaforms.Form `json:"form"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, got := range []mwanachamaforms.Form{out.Form, getFormAs(t, h, "treasurer-1", f.ID)} {
		if got.PublishedBy != "treasurer-1" || got.PublisherName != "Wanjiku Kamau" ||
			got.PublishingAuthority != "Treasurer, Kilimani Ward" {
			t.Fatalf("form = published_by %q, publisher_name %q, publishing_authority %q; want the session's caller, their directory name and the trimmed authority",
				got.PublishedBy, got.PublisherName, got.PublishingAuthority)
		}
	}

	rows, raw := waitingAs(t, h, "resident-1", "")
	if len(rows) != 1 || rows[0].PublisherName != "Wanjiku Kamau" ||
		rows[0].PublishingAuthority != "Treasurer, Kilimani Ward" {
		t.Fatalf("waiting row = %s, want the publisher's name and the authority", raw)
	}
}

func TestPublishWithNoAuthorityNamesOnlyThePublisher(t *testing.T) {
	fm := newTestManager(t)
	fm.UseNames(namesOf(map[string]string{"clerk-1": "  Otieno\nOdhiambo  "}))
	h := mountAsCaller(t, fm)
	f := composeForm(t, fm, "Water levy", "council", []string{"ward-1"}, 1)

	if rec := publishAs(t, h, "clerk-1", f.ID, `{}`); rec.Code != http.StatusOK {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := getFormAs(t, h, "clerk-1", f.ID)
	if got.PublisherName != "Otieno Odhiambo" || got.PublishingAuthority != "" {
		t.Fatalf("publisher_name %q, publishing_authority %q; want the name on one line and no authority",
			got.PublisherName, got.PublishingAuthority)
	}
}

func TestPublishWithNoNameResolverStillPublishes(t *testing.T) {
	fm := newTestManager(t)
	h := mountAsCaller(t, fm)
	f := composeForm(t, fm, "Census", "council", []string{"ward-1"}, 1)

	if rec := publishAs(t, h, "clerk-1", f.ID, `{"publishing_authority":"Ward office"}`); rec.Code != http.StatusOK {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := getFormAs(t, h, "clerk-1", f.ID)
	if got.PublisherName != "" || got.PublishingAuthority != "Ward office" || got.PublishedBy != "clerk-1" {
		t.Fatalf("form = %+v, want no name, the authority and the caller", got)
	}
}

func TestADirectoryNameIsShortenedToTheBound(t *testing.T) {
	fm := newTestManager(t)
	fm.UseNames(namesOf(map[string]string{"clerk-1": strings.Repeat("n", 500)}))
	h := mountAsCaller(t, fm)
	f := composeForm(t, fm, "Census", "council", []string{"ward-1"}, 1)

	if rec := publishAs(t, h, "clerk-1", f.ID, `{}`); rec.Code != http.StatusOK {
		t.Fatalf("publish: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := getFormAs(t, h, "clerk-1", f.ID); len([]rune(got.PublisherName)) != models.MaxPublisherName {
		t.Fatalf("publisher_name has %d characters, want %d", len([]rune(got.PublisherName)), models.MaxPublisherName)
	}
}

func TestAnAuthorityThatIsTooLongOrSpansLinesIsRefusedAndTheFormStaysUnpublished(t *testing.T) {
	for name, authority := range map[string]string{
		"too long":    strings.Repeat("a", models.MaxPublishingAuthority+1),
		"two lines":   "Treasurer\nKilimani Ward",
		"a tab in it": "Treasurer\tKilimani",
	} {
		t.Run(name, func(t *testing.T) {
			fm := newTestManager(t)
			h := mountAsCaller(t, fm)
			f := composeForm(t, fm, "Census", "council", []string{"ward-1"}, 1)

			body, _ := json.Marshal(map[string]string{"publishing_authority": authority})
			rec := publishAs(t, h, "clerk-1", f.ID, string(body))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("publish: status = %d, body = %s, want 400", rec.Code, rec.Body.String())
			}
			if got := getFormAs(t, h, "clerk-1", f.ID); got.Status != mwanachamaforms.StatusDraft || got.PublishingAuthority != "" {
				t.Fatalf("form = %+v, want an untouched draft", got)
			}
		})
	}

	fm := newTestManager(t)
	h := mountAsCaller(t, fm)
	f := composeForm(t, fm, "Census", "council", []string{"ward-1"}, 1)
	body, _ := json.Marshal(map[string]string{"publishing_authority": strings.Repeat("é", models.MaxPublishingAuthority)})
	if rec := publishAs(t, h, "clerk-1", f.ID, string(body)); rec.Code != http.StatusOK {
		t.Fatalf("an authority of exactly the bound in characters: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestComposingAFormCannotClaimAPublisherOrAnAuthority(t *testing.T) {
	fm := newTestManager(t)
	h := mountAsCaller(t, fm)
	r := httptest.NewRequest(http.MethodPost, "/forms", strings.NewReader(`{
		"title": "Census",
		"closes_at": "2099-01-01T00:00:00Z",
		"publisher_name": "Forged Name",
		"publishing_authority": "Forged Office"
	}`))
	r.Header.Set("X-Test-Caller", "clerk-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var f mwanachamaforms.Form
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := getFormAs(t, h, "clerk-1", f.ID); got.PublisherName != "" || got.PublishingAuthority != "" {
		t.Fatalf("a draft carries publisher_name %q and publishing_authority %q from its body", got.PublisherName, got.PublishingAuthority)
	}
}
