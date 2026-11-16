package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestDeclareChapter_RejectsPhantomRespondentID(t *testing.T) {
	fm := newTestManager(t)
	m := mount(t, fm)

	closesAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	f, err := fm.Create(t.Context(), mwanachamaforms.Form{
		Title: "Ward Census", OriginatorChapterID: "chapter-1",
		ClosesAt: closesAt, Audience: mwanachamaforms.AudiencePublic,
		CreatedBy: "author-1",
	})
	if err != nil {
		t.Fatalf("Create form: %v", err)
	}
	_, link, err := fm.Publish(t.Context(), f.ID, "probe-link-key", "author-1", "")
	if err != nil {
		t.Fatalf("Publish form: %v", err)
	}

	declare := func(respondentID, publicKey string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"respondent_id": respondentID, "public_key": publicKey, "declared_chapter_id": "chapter-99"})
		req := httptest.NewRequest(http.MethodPost, "/public-links/"+link.Key+"/declarations", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, req)
		return rec
	}

	if rec := declare("phantom-respondent-id", "anon-key"); rec.Code != http.StatusBadRequest {
		t.Fatalf("phantom respondent: got %d (body %s), want 400", rec.Code, rec.Body.String())
	}

	resp, err := fm.RegisterRespondent(t.Context(), link.Key, "anon-key")
	if err != nil {
		t.Fatalf("RegisterRespondent: %v", err)
	}
	if rec := declare(resp.ID, "anon-key"); rec.Code != http.StatusCreated {
		t.Fatalf("real respondent: got %d (body %s), want 201", rec.Code, rec.Body.String())
	}
}
