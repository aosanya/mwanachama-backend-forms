package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func composeAgencyWideStandingForm(t *testing.T, fm mwanachamaforms.FormManager) choiceForm {
	t.Helper()
	ctx := context.Background()
	f, err := fm.Create(ctx, mwanachamaforms.Form{
		Title:               "Raise an issue",
		OriginatorChapterID: "agency-1",
		ClosesAt:            time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		Standing:            true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "What is the issue?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return choiceForm{form: f, question: q}
}

func TestHS50AStandingFormSentToNobodyIsOpenToEveryMemberWhoIsSignedIn(t *testing.T) {
	fixtures := map[string]func(fm mwanachamaforms.FormManager){
		"no host functions": func(mwanachamaforms.FormManager) {},
		"groups and an audience that admit nobody": func(fm mwanachamaforms.FormManager) {
			fm.UseGroups(groupsOf(map[string][]string{}))
			fm.UseStandingAudience(audienceOf(map[string][]string{}))
		},
	}
	for name, wire := range fixtures {
		t.Run(name, func(t *testing.T) {
			fm := newTestManager(t)
			wire(fm)
			sf := composeAgencyWideStandingForm(t, fm)
			mux := mountReachedOnly(t, fm, true)

			for _, who := range []string{"alice", "bob"} {
				if rec := doAs(t, mux, who, http.MethodGet, "/forms/"+sf.form.ID, ""); rec.Code != http.StatusOK {
					t.Fatalf("%s get_form = %d %s, want 200", who, rec.Code, rec.Body.String())
				}
				for _, text := range []string{"first", "second"} {
					if rec := doAs(t, mux, who, http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, text)); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
						t.Fatalf("%s submit_answers = %d %s, want success", who, rec.Code, rec.Body.String())
					}
				}
			}
			if rec := doAs(t, mux, "", http.MethodGet, "/forms/"+sf.form.ID, ""); rec.Code != http.StatusNotFound {
				t.Fatalf("a caller who is not signed in: get_form = %d, want 404", rec.Code)
			}
		})
	}
}

func TestHS50EachSendToNobodyIsReadBackByItsSenderAlone(t *testing.T) {
	fm := newTestManager(t)
	sf := composeAgencyWideStandingForm(t, fm)
	mux := mountReachedOnly(t, fm, true)
	for _, text := range []string{"first", "second"} {
		if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, text)); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("submit_answers = %d %s", rec.Code, rec.Body.String())
		}
	}
	if rec := doAs(t, mux, "bob", http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, "bob's")); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("bob submit_answers = %d %s", rec.Code, rec.Body.String())
	}
	rec := doAs(t, mux, "alice", http.MethodGet, "/forms/answered", "")
	var answered []mwanachamaforms.AnsweredForm
	if err := json.Unmarshal(rec.Body.Bytes(), &answered); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("answered = %d %s", rec.Code, rec.Body.String())
	}
	if len(answered) != 2 || answered[0].SendID == answered[1].SendID {
		t.Fatalf("answered = %+v, want alice's two sends only, each its own", answered)
	}
	given := map[string]bool{}
	for _, a := range answered {
		rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sf.form.ID+"/sends/"+a.SendID, "")
		var send mwanachamaforms.Send
		if err := json.Unmarshal(rec.Body.Bytes(), &send); err != nil || rec.Code != http.StatusOK || len(send.Answers) != 1 {
			t.Fatalf("get_my_send = %d %s", rec.Code, rec.Body.String())
		}
		given[send.Answers[0].ValueText] = true
		if rec := doAs(t, mux, "bob", http.MethodGet, "/forms/"+sf.form.ID+"/sends/"+a.SendID, ""); rec.Code != http.StatusNotFound {
			t.Fatalf("bob reading alice's send = %d, want 404", rec.Code)
		}
	}
	if !given["first"] || !given["second"] {
		t.Fatalf("given = %v, want both sends kept", given)
	}
}

func TestHS50AStandingFormSentToNobodyIsNeverWaiting(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"agency-1"}}))
	composeAgencyWideStandingForm(t, fm)
	rows, body := waitingAs(t, mountAsCaller(t, fm), "alice", "")
	if len(rows) != 0 {
		t.Fatalf("waiting = %s, want nothing", body)
	}
}

func TestHS50ATargetedStandingFormStillFollowsItsAudience(t *testing.T) {
	fm, sf := standingFixture(t)
	mux := mountReachedOnly(t, fm, true)
	if rec := doAs(t, mux, "bob", http.MethodGet, "/forms/"+sf.form.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("bob get_form = %d, want 404 outside the target's audience", rec.Code)
	}
}
