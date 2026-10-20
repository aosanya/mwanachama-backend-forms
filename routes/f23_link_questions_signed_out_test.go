package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func TestASignedOutCallerReadsALinksQuestions(t *testing.T) {
	fx := newLinkFixture(t)

	rec := doAs(t, fx.mux, "", http.MethodGet, "/public-links/"+fx.linked.key+"/questions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("signed out: got %d, body %s, want 200", rec.Code, rec.Body.String())
	}
	var got []mwanachamaforms.LinkedQuestion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rec.Body.String())
	}
	if len(got) != 1 || got[0].ID != fx.linked.question.ID {
		t.Fatalf("questions = %+v, want only %q", got, fx.linked.question.ID)
	}
	if len(got[0].Options) != 2 || got[0].Options[0].Label != "Main" || got[0].Options[1].Label != "Market" {
		t.Fatalf("options = %+v, want Main then Market", got[0].Options)
	}
	for _, other := range []string{fx.other.question.ID, fx.sent.question.ID, fx.elsewhere.question.ID} {
		if strings.Contains(rec.Body.String(), other) {
			t.Fatalf("the link's questions name another form's question %q: %s", other, rec.Body.String())
		}
	}
}

func TestALinksQuestionsCarryNoAnswers(t *testing.T) {
	fx := newLinkFixture(t)
	if _, err := fx.fm.SubmitAnswers(context.Background(), fx.linked.form.ID, "alice", []mwanachamaforms.Answer{
		{QuestionID: fx.linked.question.ID, OptionIDs: []string{fx.linked.option.ID}},
	}); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}

	rec := doAs(t, fx.mux, "", http.MethodGet, "/public-links/"+fx.linked.key+"/questions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body.String())
	}
	for _, leak := range []string{"alice", "member_id", "option_ids", "answered_at"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Fatalf("the link's questions carry %q: %s", leak, rec.Body.String())
		}
	}
}

func TestALinksQuestionsAnswerAnUnknownAndAClosedLinkAlike(t *testing.T) {
	fx := newLinkFixture(t)
	if _, err := fx.fm.Close(context.Background(), fx.other.form.ID, "sender-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	unknown := doAs(t, fx.mux, "", http.MethodGet, "/public-links/no-such-link/questions", "")
	closed := doAs(t, fx.mux, "", http.MethodGet, "/public-links/"+fx.other.key+"/questions", "")
	if unknown.Code != http.StatusNotFound || closed.Code != http.StatusNotFound {
		t.Fatalf("unknown %d, closed %d, want both 404", unknown.Code, closed.Code)
	}
	if unknown.Body.String() != closed.Body.String() {
		t.Fatalf("the refusals differ: unknown %q, closed %q", unknown.Body.String(), closed.Body.String())
	}
}
