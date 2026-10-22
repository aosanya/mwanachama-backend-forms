package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

type linkedForm struct {
	choiceForm
	key string
}

func composePublicForm(t *testing.T, fm mwanachamaforms.FormManager, key string) linkedForm {
	t.Helper()
	ctx := context.Background()
	f, err := fm.Create(ctx, mwanachamaforms.Form{
		Title:               "Street lighting",
		OriginatorChapterID: "county-1",
		Audience:            mwanachamaforms.AudiencePublic,
		ClosesAt:            time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	q, opts, err := fm.AddQuestion(ctx, mwanachamaforms.QuestionDraft{
		Question: mwanachamaforms.Question{FormID: f.ID, AnswerType: mwanachamaforms.AnswerSingleChoice, Prompt: "Which street?"},
		Options:  []mwanachamaforms.QuestionOption{{Label: "Main"}, {Label: "Market"}},
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if _, _, err := fm.Publish(ctx, f.ID, key, "sender-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return linkedForm{choiceForm: choiceForm{form: f, question: q, option: opts[0]}, key: key}
}

func mountLikeAHost(t *testing.T, fm mwanachamaforms.FormManager) *http.ServeMux {
	t.Helper()
	caller := func(ctx context.Context) string {
		id, _ := ctx.Value(callerKey{}).(string)
		return id
	}
	split := routes.SplitFor(fm, routes.Mount{Caller: caller, ReachedOnly: true})
	mux := http.NewServeMux()
	for _, rt := range append(split.Anonymous, split.Gated...) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return mux
}

type linkFixture struct {
	fm        mwanachamaforms.FormManager
	mux       *http.ServeMux
	linked    linkedForm
	other     linkedForm
	sent      choiceForm
	elsewhere choiceForm
}

func newLinkFixture(t *testing.T) linkFixture {
	t.Helper()
	fm := newTestManager(t)
	fm.UseGroups(groupsOf(map[string][]string{"alice": {"ward-1"}}))
	return linkFixture{
		fm:        fm,
		mux:       mountLikeAHost(t, fm),
		linked:    composePublicForm(t, fm, "link-a"),
		other:     composePublicForm(t, fm, "link-b"),
		sent:      composeChoiceForm(t, fm, "ward-1", true),
		elsewhere: composeChoiceForm(t, fm, "ward-9", true),
	}
}

func (x linkFixture) answerThroughLink(t *testing.T, key, device, group string) string {
	t.Helper()
	rec := doAs(t, x.mux, "", http.MethodPost, "/public-links/"+key+"/respondents", `{"public_key":"`+device+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("register_respondent = %d %s", rec.Code, rec.Body.String())
	}
	var r mwanachamaforms.RespondentPublicView
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode respondent: %v", err)
	}
	rec = doAs(t, x.mux, "", http.MethodPost, "/public-links/"+key+"/declarations",
		`{"respondent_id":"`+r.ID+`","declared_chapter_id":"`+group+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("declare_chapter = %d %s", rec.Code, rec.Body.String())
	}
	return r.ID
}

func (x linkFixture) claim(t *testing.T, caller, key, device string) *http.Response {
	t.Helper()
	return doAs(t, x.mux, caller, http.MethodPost, "/public-links/"+key+"/respondents/claim", `{"public_key":"`+device+`"}`).Result()
}

func formProbes(c choiceForm, missing string) []struct{ what, method, path, missingPath, body string } {
	return []struct{ what, method, path, missingPath, body string }{
		{"get_form", http.MethodGet, "/forms/" + c.form.ID, "/forms/" + missing, ""},
		{"list_questions", http.MethodGet, "/forms/" + c.form.ID + "/questions", "/forms/" + missing + "/questions", ""},
		{"list_options", http.MethodGet, "/forms/" + c.form.ID + "/questions/" + c.question.ID + "/options", "/forms/" + missing + "/questions/no-such-question/options", ""},
		{"submit_answers", http.MethodPost, "/forms/" + c.form.ID + "/answers", "/forms/" + missing + "/answers", c.answerBody()},
	}
}

func assertHidden(t *testing.T, x linkFixture, caller, name string, c choiceForm) {
	t.Helper()
	for _, p := range formProbes(c, "no-such-form") {
		got := doAs(t, x.mux, caller, p.method, p.path, p.body)
		want := doAs(t, x.mux, caller, p.method, p.missingPath, p.body)
		if got.Code != want.Code || got.Body.String() != want.Body.String() {
			t.Errorf("%s: %s as %q = %d %q; an unknown form answers %d %q",
				name, p.what, caller, got.Code, got.Body.String(), want.Code, want.Body.String())
		}
	}
}

func TestF24ASignedInRespondentReadsAndAnswersTheFormTheirLinkOpened(t *testing.T) {
	x := newLinkFixture(t)
	x.answerThroughLink(t, x.linked.key, "token-c1", "ward-7")

	assertHidden(t, x, "carol", "before signing in, the linked form", x.linked.choiceForm)

	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("claim_respondent = %d, want 200", res.StatusCode)
	}

	if rec := doAs(t, x.mux, "carol", http.MethodGet, "/forms/"+x.linked.form.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("get_form = %d %s, want 200", rec.Code, rec.Body.String())
	}
	rec := doAs(t, x.mux, "carol", http.MethodGet, "/forms/"+x.linked.form.ID+"/questions", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), x.linked.question.ID) {
		t.Fatalf("list_questions = %d %s, want the form's question", rec.Code, rec.Body.String())
	}
	rec = doAs(t, x.mux, "carol", http.MethodGet, "/forms/"+x.linked.form.ID+"/questions/"+x.linked.question.ID+"/options", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), x.linked.option.ID) {
		t.Fatalf("list_options = %d %s, want the question's choices", rec.Code, rec.Body.String())
	}
	rec = doAs(t, x.mux, "carol", http.MethodPost, "/forms/"+x.linked.form.ID+"/answers", x.linked.answerBody())
	if rec.Code != http.StatusOK {
		t.Fatalf("submit_answers = %d %s, want 200", rec.Code, rec.Body.String())
	}
	for _, a := range decodeAnswers(t, rec) {
		if a.MemberID != "carol" || a.ChapterID != "" {
			t.Fatalf("answer filed as member %q in group %q; want carol, counted toward the form's own group (none) rather than the declared ward-7",
				a.MemberID, a.ChapterID)
		}
	}
}

func TestF24ALinkGrantsOnlyItsOwnFormAndLeavesTheGroupNarrowing(t *testing.T) {
	x := newLinkFixture(t)
	x.answerThroughLink(t, x.linked.key, "token-c1", "ward-1")
	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("claim_respondent = %d, want 200", res.StatusCode)
	}

	assertHidden(t, x, "carol", "another public form whose link carol never used", x.other.choiceForm)
	assertHidden(t, x, "carol", "a form sent to the group carol only declared", x.sent)
	assertHidden(t, x, "carol", "a form sent to another group", x.elsewhere)

	assertHidden(t, x, "dave", "the linked form, to a caller who never answered through it", x.linked.choiceForm)
	assertHidden(t, x, "alice", "the linked form, to a resident who never answered through it", x.linked.choiceForm)
	if rec := doAs(t, x.mux, "alice", http.MethodGet, "/forms/"+x.sent.form.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("alice get_form on a form sent to her group = %d, want 200", rec.Code)
	}

	for _, c := range []choiceForm{x.other.choiceForm, x.sent, x.elsewhere} {
		answers, err := x.fm.ListAllAnswers(context.Background(), c.form.ID, "carol", "")
		if err != nil {
			t.Fatalf("ListAllAnswers: %v", err)
		}
		if len(answers) != 0 {
			t.Fatalf("form %s took %d answers from carol, whose link never opened it", c.form.ID, len(answers))
		}
	}
}

func TestF24ARespondentIsClaimedOnceAndByTheCallerOnly(t *testing.T) {
	x := newLinkFixture(t)
	respondentID := x.answerThroughLink(t, x.linked.key, "token-c1", "ward-7")

	rec := doAs(t, x.mux, "carol", http.MethodPost, "/public-links/"+x.linked.key+"/respondents/claim",
		`{"public_key":"token-c1","member_id":"mallory"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("claim_respondent = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "claimed_by_member_id") || strings.Contains(rec.Body.String(), "carol") {
		t.Fatalf("the claim's answer carries who claimed it: %s", rec.Body.String())
	}
	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("carol claiming again = %d, want 200", res.StatusCode)
	}
	if res := x.claim(t, "dave", x.linked.key, "token-c1"); res.StatusCode != http.StatusConflict {
		t.Fatalf("dave claiming carol's token = %d, want 409", res.StatusCode)
	}
	assertHidden(t, x, "dave", "the linked form after a refused claim", x.linked.choiceForm)
	assertHidden(t, x, "mallory", "the linked form, to the member a claim's body named", x.linked.choiceForm)

	if res := x.claim(t, "", x.linked.key, "token-c1"); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("claiming with no caller = %d, want 400", res.StatusCode)
	}
	if res := x.claim(t, "erin", "no-such-link", "token-c1"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("claiming through an unknown link = %d, want 404", res.StatusCode)
	}
	if res := x.claim(t, "erin", x.linked.key, "no-such-device"); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("claiming an unknown token = %d, want 400", res.StatusCode)
	}

	held, err := x.fm.Declare(context.Background(), mwanachamaforms.Declaration{
		RespondentID: respondentID, FormID: x.linked.form.ID, DeclaredChapterID: "ward-7",
	})
	if err != nil || held.DeclaredChapterID != "ward-7" {
		t.Fatalf("the declaration did not survive the claim: %+v %v", held, err)
	}
}
