package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

func composeOverTheWire(t *testing.T, mux *http.ServeMux, body string) (int, mwanachamaforms.Form) {
	t.Helper()
	rec := do(t, mux, http.MethodPost, "/forms", body)
	var f mwanachamaforms.Form
	if rec.Code == http.StatusCreated {
		if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	return rec.Code, f
}

func TestAStandingFormComposedOverTheWireKeepsItsLanding(t *testing.T) {
	fm := newTestManager(t)
	mux := mount(t, fm)
	closes := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)

	code, f := composeOverTheWire(t, mux, `{"title":"Ride request","closes_at":"`+closes+`","standing":true,"landing":"thread"}`)
	if code != http.StatusCreated || f.Landing != mwanachamaforms.LandingThread || !f.Standing || !f.WindowAdvisory {
		t.Fatalf("create: %d %+v", code, f)
	}
	rec := do(t, mux, http.MethodGet, "/forms/"+f.ID, "")
	var read map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &read); err != nil || read["landing"] != "thread" {
		t.Fatalf("read: %d %s", rec.Code, rec.Body.String())
	}
	rec = do(t, mux, http.MethodPatch, "/forms/"+f.ID, `{"title":"Ride request","closes_at":"`+closes+`","audience":"member","collection_mode":"self","landing":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	got, err := fm.Get(context.Background(), f.ID)
	if err != nil || got.Landing != mwanachamaforms.LandingThread {
		t.Fatalf("after an edit naming no landing: %+v, %v", got, err)
	}
}

func TestALandingTheFormCannotHaveIsABadRequestOverTheWire(t *testing.T) {
	mux := mount(t, newTestManager(t))
	closes := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	for _, body := range []string{
		`{"title":"x","closes_at":"` + closes + `","landing":"thread"}`,
		`{"title":"x","closes_at":"` + closes + `","standing":true,"landing":"email"}`,
	} {
		rec := do(t, mux, http.MethodPost, "/forms", body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s, want 400", body, rec.Code, rec.Body.String())
		}
	}
}

func TestTheComposeToolOffersStandingAndALanding(t *testing.T) {
	tools, err := routes.BuildGatedTools(newTestManager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if tool.Action != "forms.form.create" {
			continue
		}
		var schema struct {
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("schema: %v", err)
		}
		if got := schema.Properties["landing"].Type; got != "string" {
			t.Fatalf("landing in the compose tool is %q, want a string", got)
		}
		if got := schema.Properties["standing"].Type; got != "boolean" {
			t.Fatalf("standing in the compose tool is %q, want a boolean", got)
		}
		return
	}
	t.Fatal("no compose tool built")
}

func TestTheSubmitResponseNamesTheSendTheSendAddressReadsBack(t *testing.T) {
	fm, sf := standingFixture(t)
	mux := mountReachedOnly(t, fm, true)
	rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, "need a ride"))
	var stored []mwanachamaforms.Answer
	if err := json.Unmarshal(rec.Body.Bytes(), &stored); err != nil || len(stored) != 1 || stored[0].SendID == "" {
		t.Fatalf("submit_answers = %d %s, want one answer carrying send_id", rec.Code, rec.Body.String())
	}
	rec = doAs(t, mux, "alice", http.MethodGet, "/forms/"+sf.form.ID+"/sends/"+stored[0].SendID, "")
	var send mwanachamaforms.Send
	if err := json.Unmarshal(rec.Body.Bytes(), &send); err != nil || rec.Code != http.StatusOK || send.Submission.ID != stored[0].SendID {
		t.Fatalf("get_my_send = %d %s", rec.Code, rec.Body.String())
	}
}
