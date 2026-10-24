package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func audienceOf(byGroup map[string][]string) mwanachamaforms.StandingAudience {
	return func(_ context.Context, callerID, groupID string) (bool, error) {
		for _, id := range byGroup[groupID] {
			if id == callerID {
				return true, nil
			}
		}
		return false, nil
	}
}

func composeStandingForm(t *testing.T, fm mwanachamaforms.FormManager, target string) choiceForm {
	t.Helper()
	ctx := context.Background()
	f, err := fm.Create(ctx, mwanachamaforms.Form{
		Title:               "Apply",
		OriginatorChapterID: "county-1",
		ClosesAt:            time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		Standing:            true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
		FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: "What do you need?",
	}})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, err := fm.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: target}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return choiceForm{form: f, question: q}
}

func standingBody(c choiceForm, text string) string {
	return `{"answers":[{"question_id":"` + c.question.ID + `","value_text":"` + text + `"}]}`
}

func standingFixture(t *testing.T) (mwanachamaforms.FormManager, choiceForm) {
	t.Helper()
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}, "bob": {"service-1"}, "carol": {}}))
	fm.UseStandingAudience(audienceOf(map[string][]string{"service-1": {"alice", "carol"}}))
	return fm, composeStandingForm(t, fm, "service-1")
}

func TestF91AStandingFormIsReachedByTheAudienceTheHostSupplies(t *testing.T) {
	fm, sf := standingFixture(t)
	mux := mountReachedOnly(t, fm, true)

	for _, who := range []string{"alice", "carol"} {
		if rec := doAs(t, mux, who, http.MethodGet, "/forms/"+sf.form.ID, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s get_form = %d %s, want 200 through the audience", who, rec.Code, rec.Body.String())
		}
		if rec := doAs(t, mux, who, http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, "help")); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("%s submit_answers = %d %s, want success", who, rec.Code, rec.Body.String())
		}
	}
	if rec := doAs(t, mux, "bob", http.MethodGet, "/forms/"+sf.form.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("bob belongs to the target group but is outside its audience: get_form = %d, want 404", rec.Code)
	}
	if rec := doAs(t, mux, "bob", http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, "help")); rec.Code != http.StatusNotFound {
		t.Fatalf("bob submit_answers = %d %s, want 404", rec.Code, rec.Body.String())
	}
	sends, _ := fm.ListSubmissions(context.Background(), sf.form.ID, "carol", "")
	if len(sends) != 1 || sends[0].ChapterID != "service-1" {
		t.Fatalf("carol's send = %+v, want one recorded as the group whose audience admitted her", sends)
	}
}

func TestF91AStandingFormWithNoAudienceReachesNobody(t *testing.T) {
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"service-1"}}))
	sf := composeStandingForm(t, fm, "service-1")
	mux := mountReachedOnly(t, fm, true)
	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sf.form.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get_form with no audience supplied = %d, want 404", rec.Code)
	}
}

func TestF91AStandingFormIsNeverWaiting(t *testing.T) {
	fm, _ := standingFixture(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"service-1"}}))
	ordinary := composeForm(t, fm, "Survey", "county-1", []string{"service-1"}, 1)
	if _, _, err := fm.Publish(context.Background(), ordinary.ID, "", "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	rows, body := waitingAs(t, mountAsCaller(t, fm), "alice", "")
	if len(rows) != 1 || rows[0].FormID != ordinary.ID {
		t.Fatalf("waiting = %s, want only the ordinary form", body)
	}
}

func TestF91EverySendIsItsOwnRecordReadBackByIdByItsSenderAlone(t *testing.T) {
	fm, sf := standingFixture(t)
	mux := mountReachedOnly(t, fm, true)
	for _, text := range []string{"first", "second"} {
		if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sf.form.ID+"/answers", standingBody(sf, text)); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("submit_answers = %d %s", rec.Code, rec.Body.String())
		}
	}
	rec := doAs(t, mux, "alice", http.MethodGet, "/forms/answered", "")
	var answered []mwanachamaforms.AnsweredForm
	if err := json.Unmarshal(rec.Body.Bytes(), &answered); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("answered = %d %s", rec.Code, rec.Body.String())
	}
	given := map[string]bool{}
	for _, a := range answered {
		if a.FormID != sf.form.ID || a.SendID == "" {
			t.Fatalf("answered row %+v carries no send id", a)
		}
		rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sf.form.ID+"/sends/"+a.SendID, "")
		var send mwanachamaforms.Send
		if err := json.Unmarshal(rec.Body.Bytes(), &send); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("get_my_send = %d %s", rec.Code, rec.Body.String())
		}
		if send.Submission.ID != a.SendID || len(send.Answers) != 1 || send.Submission.Answers != 1 {
			t.Fatalf("send = %+v, want the one send with its one answer", send)
		}
		given[send.Answers[0].ValueText] = true

		if rec := doAs(t, mux, "carol", http.MethodGet, "/forms/"+sf.form.ID+"/sends/"+a.SendID, ""); rec.Code != http.StatusNotFound {
			t.Fatalf("carol reading alice's send = %d %s, want 404", rec.Code, rec.Body.String())
		}
	}
	if len(answered) != 2 || !given["first"] || !given["second"] {
		t.Fatalf("answered = %+v given = %v, want two sends, neither overwritten", answered, given)
	}
	if rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sf.form.ID+"/sends/no-such-send", ""); rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "GetSend") {
		t.Fatalf("unknown send = %d %s, want a bare 404", rec.Code, rec.Body.String())
	}
}
