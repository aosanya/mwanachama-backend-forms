package routes_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestAResidentReadingTheirSendDoesNotReadTheMemberWhoMarkedItPaid(t *testing.T) {
	fm, sent, _, _ := residentFixture(t)
	mux := mountReachedOnly(t, fm, true)
	if rec := doAs(t, mux, "alice", http.MethodPost, "/forms/"+sent.form.ID+"/answers", sent.answerBody()); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("submit_answers = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := fm.MarkPaid(context.Background(), sent.form.ID, "alice", true, "coordinator-77"); err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}

	rec := doAs(t, mux, "alice", http.MethodGet, "/forms/"+sent.form.ID+"/submissions/mine", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get_my_submission = %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "coordinator-77") {
		t.Fatalf("a resident reads the member who marked the send paid: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"paid":true`) {
		t.Fatalf("the paid mark itself is lost: %s", rec.Body.String())
	}
}
