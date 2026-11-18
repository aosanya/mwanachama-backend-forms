package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-forms/routes"
	"github.com/aosanya/mwanachama-backend-shared/consent"
)

type clerkDesk struct {
	fm       mwanachamaforms.FormManager
	mux      *http.ServeMux
	form     mwanachamaforms.Form
	required mwanachamaforms.Question
	optional mwanachamaforms.Question
}

type consentOf map[string]error

func (c consentOf) Check(_ context.Context, actorID string) error { return c[actorID] }

func newClerkDesk(t *testing.T) clerkDesk {
	t.Helper()
	fm := newTestManager(t)
	ctx := context.Background()
	f := seedForm(t, fm)
	ask := func(prompt string, required bool) mwanachamaforms.Question {
		q, _, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{Question: mwanachamaforms.Question{
			FormID: f.ID, AnswerType: mwanachamaforms.AnswerFreeText, Prompt: prompt, Required: required,
		}})
		if err != nil {
			t.Fatalf("AddQuestion %q: %v", prompt, err)
		}
		return q
	}
	required, optional := ask("Name of the stall", true), ask("Anything else", false)
	if _, err := fm.AddTarget(ctx, mwanachamaforms.Target{FormID: f.ID, ChapterID: "desk"}); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, _, err := fm.Publish(ctx, f.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	fm.UseGroups(func(_ context.Context, memberID string) ([]string, error) {
		if memberID == "resident-1" || memberID == "resident-2" {
			return []string{"desk"}, nil
		}
		return nil, nil
	})
	return clerkDesk{fm: fm, mux: callerMount(t, fm), form: f, required: required, optional: optional}
}

func routesWithReachedOnly(fm mwanachamaforms.FormManager) []routes.Route {
	caller := func(ctx context.Context) string {
		id, _ := ctx.Value(callerKey{}).(string)
		return id
	}
	return routes.RoutesFor(fm, routes.Mount{Caller: caller, ReachedOnly: true})
}

func (d clerkDesk) filePath(memberID, query string) string {
	return "/forms/" + d.form.ID + "/submissions/" + memberID + "/answers" + query
}

func (d clerkDesk) answersBody(stall string) string {
	return `{"answers":[{"question_id":"` + d.required.ID + `","value_text":"` + stall + `"}]}`
}

func (d clerkDesk) file(t *testing.T, caller, memberID, query, stall string) *httptest.ResponseRecorder {
	t.Helper()
	return doAs(t, d.mux, caller, http.MethodPost, d.filePath(memberID, query), d.answersBody(stall))
}

func (d clerkDesk) sendsOf(t *testing.T, memberID string) []mwanachamaforms.Submission {
	t.Helper()
	rec := doAs(t, d.mux, "clerk-1", http.MethodGet, "/forms/"+d.form.ID+"/submissions?member_id="+memberID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list sends of %s: %d %s", memberID, rec.Code, rec.Body)
	}
	var out []mwanachamaforms.Submission
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode sends: %v (body %s)", err, rec.Body)
	}
	return out
}

func TestMUN1373AClerkFilesAnApplicationForAResidentInTheDeskGroup(t *testing.T) {
	d := newClerkDesk(t)

	rec := d.file(t, "clerk-1", "resident-1", "", "Mama Zawadi's mandazi")
	if rec.Code != http.StatusOK {
		t.Fatalf("file: %d %s, want 200", rec.Code, rec.Body)
	}
	given := decodeAnswers(t, rec)
	if len(given) != 1 || given[0].MemberID != "resident-1" || given[0].SendID == "" {
		t.Fatalf("answers = %+v, want one answer of resident-1 carrying the send id", given)
	}

	sends := d.sendsOf(t, "resident-1")
	if len(sends) != 1 {
		t.Fatalf("sends of resident-1 = %+v, want one", sends)
	}
	got := sends[0]
	if got.MemberID != "resident-1" || got.ChapterID != "desk" || got.Channel != models.ChannelPaper || got.Collector != "clerk-1" || got.CollectedAt == "" {
		t.Fatalf("send = %+v, want resident-1 in desk on the paper channel collected by clerk-1", got)
	}
	if got.ID != given[0].SendID {
		t.Fatalf("send id = %q, answer send_id = %q, want the same", got.ID, given[0].SendID)
	}
	if clerks := d.sendsOf(t, "clerk-1"); len(clerks) != 0 {
		t.Fatalf("sends of the clerk = %+v, want none", clerks)
	}
	held, err := d.fm.ListAnswers(context.Background(), d.form.ID, "resident-1")
	if err != nil || len(held) != 1 || held[0].ValueText != "Mama Zawadi's mandazi" {
		t.Fatalf("answers of resident-1 = %+v (%v), want the filed answer", held, err)
	}
}

func TestMUN1373AFieldWorkerFilesOnTheFieldChannel(t *testing.T) {
	d := newClerkDesk(t)

	if rec := d.file(t, "clerk-1", "resident-1", "?channel=field", "Stall A"); rec.Code != http.StatusOK {
		t.Fatalf("file on field: %d %s, want 200", rec.Code, rec.Body)
	}
	sends := d.sendsOf(t, "resident-1")
	if len(sends) != 1 || sends[0].Channel != models.ChannelField {
		t.Fatalf("sends = %+v, want one on the field channel", sends)
	}
}

func TestMUN1373FilingTwiceForTheSamePersonEditsTheOneSend(t *testing.T) {
	d := newClerkDesk(t)

	first := decodeAnswers(t, d.file(t, "clerk-1", "resident-1", "", "Stall A"))
	second := d.file(t, "clerk-2", "resident-1", "", "Stall B")
	if second.Code != http.StatusOK {
		t.Fatalf("file again: %d %s, want 200", second.Code, second.Body)
	}
	again := decodeAnswers(t, second)
	if len(first) != 1 || len(again) != 1 || first[0].SendID != again[0].SendID {
		t.Fatalf("send ids %+v then %+v, want the same send", first, again)
	}
	sends := d.sendsOf(t, "resident-1")
	if len(sends) != 1 || sends[0].EditedAt == "" || sends[0].Collector != "clerk-2" {
		t.Fatalf("sends = %+v, want one send, edited, last collected by clerk-2", sends)
	}
	held, err := d.fm.ListAnswers(context.Background(), d.form.ID, "resident-1")
	if err != nil || len(held) != 1 || held[0].ValueText != "Stall B" {
		t.Fatalf("answers = %+v (%v), want the one answer rewritten", held, err)
	}
}

func TestMUN1373ASendIsFiledForTheNamedPersonNotForTheNextOne(t *testing.T) {
	d := newClerkDesk(t)

	if rec := d.file(t, "clerk-1", "resident-1", "", "Stall A"); rec.Code != http.StatusOK {
		t.Fatalf("file for resident-1: %d %s", rec.Code, rec.Body)
	}
	if rec := d.file(t, "clerk-1", "resident-2", "", "Stall B"); rec.Code != http.StatusOK {
		t.Fatalf("file for resident-2: %d %s", rec.Code, rec.Body)
	}
	for _, id := range []string{"resident-1", "resident-2"} {
		if sends := d.sendsOf(t, id); len(sends) != 1 || sends[0].MemberID != id {
			t.Fatalf("sends of %s = %+v, want exactly their own", id, sends)
		}
	}
}

func TestMUN1373APersonOutsideTheDeskGroupIsRefusedAndNothingIsKept(t *testing.T) {
	d := newClerkDesk(t)

	rec := d.file(t, "clerk-1", "stranger-1", "", "Stall A")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("file for a stranger: %d %s, want 400", rec.Code, rec.Body)
	}
	if sends := d.sendsOf(t, "stranger-1"); len(sends) != 0 {
		t.Fatalf("sends of the stranger = %+v, want none", sends)
	}
	if held, _ := d.fm.ListAnswers(context.Background(), d.form.ID, "stranger-1"); len(held) != 0 {
		t.Fatalf("answers of the stranger = %+v, want none", held)
	}
}

func TestMUN1373TheClerkBeingInTheDeskGroupDoesNotLetAStrangerThrough(t *testing.T) {
	d := newClerkDesk(t)
	d.fm.UseGroups(func(_ context.Context, memberID string) ([]string, error) {
		if memberID == "resident-1" || memberID == "clerk-1" {
			return []string{"desk"}, nil
		}
		return nil, nil
	})

	if rec := d.file(t, "clerk-1", "stranger-1", "", "Stall A"); rec.Code != http.StatusBadRequest {
		t.Fatalf("file for a stranger by a clerk in the desk group: %d %s, want 400", rec.Code, rec.Body)
	}
}

func TestMUN1373ARouteMountedForReachedFormsStillChecksTheNamedPerson(t *testing.T) {
	d := newClerkDesk(t)
	mux := http.NewServeMux()
	for _, rt := range routesWithReachedOnly(d.fm) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}

	if rec := doAs(t, mux, "clerk-1", http.MethodPost, d.filePath("resident-1", ""), d.answersBody("Stall A")); rec.Code != http.StatusOK {
		t.Fatalf("file for a resident, the clerk being outside the desk group: %d %s, want 200", rec.Code, rec.Body)
	}
	if rec := doAs(t, mux, "clerk-1", http.MethodPost, d.filePath("stranger-1", ""), d.answersBody("Stall A")); rec.Code != http.StatusBadRequest {
		t.Fatalf("file for a stranger: %d %s, want 400", rec.Code, rec.Body)
	}
}

func TestMUN1373ABadChannelIsRefusedWhateverItIs(t *testing.T) {
	d := newClerkDesk(t)

	for _, channel := range []string{"app", "link", "sms", "PAPER"} {
		if rec := d.file(t, "clerk-1", "resident-1", "?channel="+channel, "Stall A"); rec.Code != http.StatusBadRequest {
			t.Fatalf("file on %q: %d %s, want 400", channel, rec.Code, rec.Body)
		}
	}
	if sends := d.sendsOf(t, "resident-1"); len(sends) != 0 {
		t.Fatalf("sends = %+v, want none after refused channels", sends)
	}
}

func TestMUN1373AMissingRequiredAnswerIsRefusedAndNothingIsKept(t *testing.T) {
	d := newClerkDesk(t)

	body := `{"answers":[{"question_id":"` + d.optional.ID + `","value_text":"only this"}]}`
	rec := doAs(t, d.mux, "clerk-1", http.MethodPost, d.filePath("resident-1", ""), body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("file without the required answer: %d %s, want 400", rec.Code, rec.Body)
	}
	if sends := d.sendsOf(t, "resident-1"); len(sends) != 0 {
		t.Fatalf("sends = %+v, want none", sends)
	}
}

func TestMUN1373FilingWithNoCallerIsRefused(t *testing.T) {
	d := newClerkDesk(t)

	if rec := d.file(t, "", "resident-1", "", "Stall A"); rec.Code != http.StatusBadRequest {
		t.Fatalf("file with no caller: %d %s, want 400", rec.Code, rec.Body)
	}
	if sends := d.sendsOf(t, "resident-1"); len(sends) != 0 {
		t.Fatalf("sends = %+v, want none", sends)
	}
}

func TestMUN1373TheConsentGateChecksTheNamedPersonAndNotTheClerk(t *testing.T) {
	d := newClerkDesk(t)
	d.fm.UseConsentGate(consentOf{"resident-1": consent.ErrRequired, "resident-2": consent.ErrWithdrawn, "clerk-1": consent.ErrRequired})

	if rec := d.file(t, "clerk-1", "resident-1", "", "Stall A"); rec.Code != http.StatusForbidden {
		t.Fatalf("file for a person with no consent: %d %s, want 403", rec.Code, rec.Body)
	}
	if rec := d.file(t, "clerk-1", "resident-2", "", "Stall B"); rec.Code != http.StatusForbidden {
		t.Fatalf("file for a person who withdrew: %d %s, want 403", rec.Code, rec.Body)
	}
	for _, id := range []string{"resident-1", "resident-2"} {
		if sends := d.sendsOf(t, id); len(sends) != 0 {
			t.Fatalf("sends of %s = %+v, want none", id, sends)
		}
	}
}

func TestMUN1373ThePersonsConsentLetsAClerkWhoHasNoneFile(t *testing.T) {
	d := newClerkDesk(t)
	d.fm.UseConsentGate(consentOf{"clerk-1": consent.ErrRequired})

	if rec := d.file(t, "clerk-1", "resident-1", "", "Stall A"); rec.Code != http.StatusOK {
		t.Fatalf("file for a consenting person by a clerk with no consent: %d %s, want 200", rec.Code, rec.Body)
	}
}

func TestMUN1373AResidentAnsweringForThemselvesIsStillCollectedByThemselvesOnTheAppChannel(t *testing.T) {
	d := newClerkDesk(t)

	rec := doAs(t, d.mux, "resident-1", http.MethodPost, "/forms/"+d.form.ID+"/answers", d.answersBody("Stall A"))
	if rec.Code != http.StatusOK {
		t.Fatalf("answer: %d %s, want 200", rec.Code, rec.Body)
	}
	sends := d.sendsOf(t, "resident-1")
	if len(sends) != 1 || sends[0].Channel != models.ChannelApp || sends[0].Collector != "resident-1" {
		t.Fatalf("sends = %+v, want one collected by resident-1 on the app channel", sends)
	}
}
