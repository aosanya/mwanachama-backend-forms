package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
)

func (x linkFixture) registerThroughLink(t *testing.T, key, device string) string {
	t.Helper()
	rec := doAs(t, x.mux, "", http.MethodPost, "/public-links/"+key+"/respondents", `{"public_key":"`+device+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("register_respondent = %d %s", rec.Code, rec.Body.String())
	}
	var r mwanachamaforms.RespondentPublicView
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode respondent: %v", err)
	}
	return r.ID
}

func assertOpen(t *testing.T, x linkFixture, caller, name string, c choiceForm) {
	t.Helper()
	for _, p := range formProbes(c, "no-such-form") {
		if rec := doAs(t, x.mux, caller, p.method, p.path, p.body); rec.Code != http.StatusOK {
			t.Errorf("%s: %s as %q = %d %s, want 200", name, p.what, caller, rec.Code, rec.Body.String())
		}
	}
}

func TestF29AClaimedLinkGrantsItsFormWithNoDeclaration(t *testing.T) {
	x := newLinkFixture(t)
	x.registerThroughLink(t, x.linked.key, "token-c1")

	assertHidden(t, x, "carol", "the linked form before the claim", x.linked.choiceForm)
	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("claim_respondent = %d, want 200", res.StatusCode)
	}

	assertOpen(t, x, "carol", "the linked form, claimed with nothing declared", x.linked.choiceForm)
	answers, err := x.fm.ListAllAnswers(context.Background(), x.linked.form.ID, "carol")
	if err != nil {
		t.Fatalf("ListAllAnswers: %v", err)
	}
	if len(answers) == 0 {
		t.Fatal("carol's answer through the claimed link was not recorded")
	}
	for _, a := range answers {
		if a.MemberID != "carol" || a.ChapterID != "" {
			t.Fatalf("answer filed as member %q in group %q; want carol under the form's own group (none)", a.MemberID, a.ChapterID)
		}
	}

	assertHidden(t, x, "carol", "another public form whose link carol never claimed through", x.other.choiceForm)
	assertHidden(t, x, "carol", "a form sent to a group carol is not in", x.elsewhere)
	assertHidden(t, x, "dave", "the linked form, to a caller who claimed nothing", x.linked.choiceForm)
}

func TestF29ATokenClaimedThroughASecondLinkGrantsThatFormToo(t *testing.T) {
	x := newLinkFixture(t)
	x.registerThroughLink(t, x.linked.key, "token-c1")
	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("first claim = %d, want 200", res.StatusCode)
	}
	assertHidden(t, x, "carol", "the second link's form before claiming through it", x.other.choiceForm)

	x.registerThroughLink(t, x.other.key, "token-c1")
	if res := x.claim(t, "carol", x.other.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("claim through the second link = %d, want 200", res.StatusCode)
	}
	assertOpen(t, x, "carol", "the second link's form", x.other.choiceForm)

	if res := x.claim(t, "dave", x.other.key, "token-c1"); res.StatusCode != http.StatusConflict {
		t.Fatalf("dave claiming carol's token through the second link = %d, want 409", res.StatusCode)
	}
	assertHidden(t, x, "dave", "the second link's form after a refused claim", x.other.choiceForm)
}

func TestF29ADeclarationIsStillRecordedBesideTheClaim(t *testing.T) {
	x := newLinkFixture(t)
	respondentID := x.registerThroughLink(t, x.linked.key, "token-c1")
	if res := x.claim(t, "carol", x.linked.key, "token-c1"); res.StatusCode != http.StatusOK {
		t.Fatalf("claim_respondent = %d, want 200", res.StatusCode)
	}
	rec := doAs(t, x.mux, "", http.MethodPost, "/public-links/"+x.linked.key+"/declarations",
		`{"respondent_id":"`+respondentID+`","declared_chapter_id":"ward-7"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("declare_chapter after the claim = %d %s, want 201", rec.Code, rec.Body.String())
	}
	var held mwanachamaforms.Declaration
	if err := json.Unmarshal(rec.Body.Bytes(), &held); err != nil || held.DeclaredChapterID != "ward-7" || held.FormID != x.linked.form.ID {
		t.Fatalf("the declaration was not recorded as made: %s %v", rec.Body.String(), err)
	}
	assertOpen(t, x, "carol", "the linked form, claimed and declared", x.linked.choiceForm)
	assertHidden(t, x, "carol", "a form sent to the group carol only declared", x.sent)
}
