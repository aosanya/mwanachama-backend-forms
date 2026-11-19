package mwanachamaforms_test

import (
	"context"
	"encoding/json"
	"errors"
	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-forms/routes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecidingApprovedSetsTheFourFieldsAndTheApplicantReadsThemBack(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	before := models.NowRFC3339()
	decided, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "inspection passed", "clerk-1")
	if err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	if decided.Decision != models.DecisionApproved || decided.DecidedBy != "clerk-1" || decided.DecisionNote != "inspection passed" || decided.DecidedAt < before {
		t.Fatalf("decided send = %+v", decided)
	}
	mine, err := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if err != nil {
		t.Fatalf("ShowMySubmission: %v", err)
	}
	if mine.Decision != models.DecisionApproved || mine.DecidedBy != "clerk-1" || mine.DecisionNote != "inspection passed" || mine.DecidedAt != decided.DecidedAt {
		t.Fatalf("read_mine = %+v, want the decision as recorded", mine)
	}
	if mine.PaidBy != "" {
		t.Fatalf("read_mine paid_by = %q, want it cleared as before", mine.PaidBy)
	}
}
func TestTheApplicantReadsTheDecisionOnTheirSendByItsId(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionRefused, "no plan attached", "clerk-1"); err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	rows, err := r.m.ListAnswered(ctx, "mem-1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListAnswered = %+v (%v)", rows, err)
	}
	send, err := r.m.GetSend(ctx, r.form.ID, rows[0].SendID, "mem-1")
	if err != nil {
		t.Fatalf("GetSend: %v", err)
	}
	if send.Submission.Decision != models.DecisionRefused || send.Submission.DecisionNote != "no plan attached" || send.Submission.DecidedBy != "clerk-1" {
		t.Fatalf("read_my_send submission = %+v", send.Submission)
	}
}
func TestARefusalWithoutANoteIsRefusedAndLeavesTheSendUndecided(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	for _, note := range []string{"", "   "} {
		if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionRefused, note, "clerk-1"); !errors.Is(err, forms.ErrMissingNote) {
			t.Fatalf("refusal with note %q = %v, want ErrMissingNote", note, err)
		}
	}
	mine, err := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if err != nil || mine.Decision != "" {
		t.Fatalf("send after the refused refusal = %+v (%v), want undecided", mine, err)
	}
}
func TestARefusalWithANoteIsStored(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	got, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionRefused, "fence taller than allowed", "clerk-1")
	if err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	if got.Decision != models.DecisionRefused || got.DecisionNote != "fence taller than allowed" || got.DecidedBy != "clerk-1" || got.DecidedAt == "" {
		t.Fatalf("refused send = %+v", got)
	}
	mine, _ := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if mine.Decision != models.DecisionRefused || mine.DecisionNote != "fence taller than allowed" {
		t.Fatalf("read_mine = %+v", mine)
	}
}
func TestAnApprovalNeedsNoNote(t *testing.T) {
	r := sentOnce(t)
	got, err := r.m.DecideSubmission(context.Background(), r.form.ID, "mem-1", models.DecisionApproved, "", "clerk-1")
	if err != nil || got.Decision != models.DecisionApproved || got.DecisionNote != "" {
		t.Fatalf("approval without note = %+v (%v)", got, err)
	}
}
func TestTheApplicantCannotDecideTheirOwnSend(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "", "mem-1"); !errors.Is(err, forms.ErrSelfDecision) {
		t.Fatalf("own send decided by the applicant = %v, want ErrSelfDecision", err)
	}
	mine, _ := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if mine.Decision != "" {
		t.Fatalf("send after the refused self decision = %+v, want undecided", mine)
	}
}
func TestDecidingAnUnknownSendIsNotFound(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-9", models.DecisionApproved, "", "clerk-1"); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("nobody sent = %v, want ErrSubmissionNotFound", err)
	}
	if _, err := r.m.DecideSubmission(ctx, "no-such-form", "mem-1", models.DecisionApproved, "", "clerk-1"); err == nil {
		t.Fatal("unknown form decided without error")
	}
}
func TestADecisionOfAWithdrawnSendIsNotFound(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if err := r.m.HideMemberAnswers(ctx, "mem-1"); err != nil {
		t.Fatalf("HideMemberAnswers: %v", err)
	}
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "", "clerk-1"); !errors.Is(err, forms.ErrSubmissionNotFound) {
		t.Fatalf("withdrawn send = %v, want ErrSubmissionNotFound", err)
	}
}
func TestADecisionMustBeApprovedOrRefusedAndANeedsADecider(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	for _, word := range []string{"", "maybe", "Approved"} {
		if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.Decision(word), "n", "clerk-1"); !errors.Is(err, forms.ErrInvalidDecision) {
			t.Fatalf("decision %q = %v, want ErrInvalidDecision", word, err)
		}
	}
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "", ""); !errors.Is(err, forms.ErrInvalidDecision) {
		t.Fatalf("nobody deciding = %v, want ErrInvalidDecision", err)
	}
}
func TestDecidingAgainReplacesTheEarlierDecisionAndItsNote(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionRefused, "plan missing", "clerk-1"); err != nil {
		t.Fatalf("first decision: %v", err)
	}
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "", "clerk-2"); err != nil {
		t.Fatalf("second decision: %v", err)
	}
	mine, err := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if err != nil {
		t.Fatalf("ShowMySubmission: %v", err)
	}
	if mine.Decision != models.DecisionApproved || mine.DecidedBy != "clerk-2" || mine.DecisionNote != "" {
		t.Fatalf("send after deciding again = %+v, want the second decision alone", mine)
	}
	sent, err := r.m.ListSubmissions(ctx, r.form.ID, "mem-1", "", "")
	if err != nil || len(sent) != 1 {
		t.Fatalf("sends after deciding twice = %+v (%v), want one", sent, err)
	}
}
func TestAnUndecidedSendReadsWithAnEmptyDecision(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	mine, err := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if err != nil {
		t.Fatalf("ShowMySubmission: %v", err)
	}
	if mine.Decision != "" || mine.DecidedBy != "" || mine.DecidedAt != "" || mine.DecisionNote != "" {
		t.Fatalf("undecided send = %+v", mine)
	}
}
func TestTheAnsweredListRowCarriesTheDecisionAndWhenItWasMade(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	rows, err := r.m.ListAnswered(ctx, "mem-1")
	if err != nil || len(rows) != 1 || rows[0].Decision != "" || rows[0].DecidedAt != "" {
		t.Fatalf("answered before deciding = %+v (%v), want one undecided row", rows, err)
	}
	decided, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "", "clerk-1")
	if err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	rows, err = r.m.ListAnswered(ctx, "mem-1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("answered after deciding = %+v (%v)", rows, err)
	}
	if rows[0].Decision != models.DecisionApproved || rows[0].DecidedAt != decided.DecidedAt {
		t.Fatalf("answered row = %+v, want approved at %s", rows[0], decided.DecidedAt)
	}
}
func TestADecisionStaysAfterTheApplicantSendsAgainOrAClerkCorrectsIt(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if _, err := r.m.DecideSubmission(ctx, r.form.ID, "mem-1", models.DecisionApproved, "ok", "clerk-1"); err != nil {
		t.Fatalf("DecideSubmission: %v", err)
	}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "clerk-1", []models.Answer{{QuestionID: r.question.ID, ValueText: "street lighting"}}); err != nil {
		t.Fatalf("CorrectSubmission: %v", err)
	}
	mine, err := r.m.ShowMySubmission(ctx, r.form.ID, "mem-1")
	if err != nil || mine.Decision != models.DecisionApproved || mine.DecisionNote != "ok" {
		t.Fatalf("send after a correction = %+v (%v), want the decision kept", mine, err)
	}
}

const (
	farDay      = "2099-12-31"
	pastDay     = "2020-01-01"
	certificate = "PRM-2026-0417"
)

func callRoute(t *testing.T, r voiceRound, caller, action, body string, path map[string]string) (int, []byte) {
	t.Helper()
	mount := routes.Mount{Caller: func(context.Context) string {
		return caller
	}}
	for _, route := range routes.RoutesFor(r.m, mount) {
		if route.Action != action {
			continue
		}
		req := httptest.NewRequest(route.Method, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("form_id", r.form.ID)
		for key, value := range path {
			req.SetPathValue(key, value)
		}
		rec := httptest.NewRecorder()
		route.Handler(rec, req)
		return rec.Code, rec.Body.Bytes()
	}
	t.Fatalf("no route for %s", action)
	return 0, nil
}
func decideOverTheRoute(t *testing.T, r voiceRound, body string) (int, []byte) {
	t.Helper()
	return callRoute(t, r, "clerk-1", "forms.submission.decide", body, map[string]string{"member_id": "mem-1"})
}
func fieldsOf(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("not a JSON object: %s", raw)
	}
	return got
}
func councilFieldsOf(t *testing.T, r voiceRound) map[string]any {
	t.Helper()
	status, raw := callRoute(t, r, "clerk-1", "forms.submission.list", "", nil)
	var rows []map[string]any
	if status != http.StatusOK || json.Unmarshal(raw, &rows) != nil || len(rows) != 1 {
		t.Fatalf("council list = %d %s, want one send", status, raw)
	}
	return rows[0]
}
func readMineFieldsOf(t *testing.T, r voiceRound) map[string]any {
	t.Helper()
	status, raw := callRoute(t, r, "mem-1", "forms.submission.read_mine", "", nil)
	if status != http.StatusOK {
		t.Fatalf("read_mine = %d %s", status, raw)
	}
	return fieldsOf(t, raw)
}
func readMySendFieldsOf(t *testing.T, r voiceRound) map[string]any {
	t.Helper()
	rows, err := r.m.ListAnswered(context.Background(), "mem-1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListAnswered = %+v (%v)", rows, err)
	}
	status, raw := callRoute(t, r, "mem-1", "forms.submission.read_my_send", "", map[string]string{"send_id": rows[0].SendID})
	if status != http.StatusOK {
		t.Fatalf("read_my_send = %d %s", status, raw)
	}
	submission, _ := fieldsOf(t, raw)["submission"].(map[string]any)
	return submission
}
func grantBody(extra string) string {
	return `{"decision":"approved"` + extra + `}`
}
func TestAGrantKeepsTheCertificateAndTheDayItEndsAndTheDecisionResponseCarriesThem(t *testing.T) {
	r := sentOnce(t)
	status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`))
	if status != http.StatusOK {
		t.Fatalf("grant with certificate = %d %s", status, raw)
	}
	got := fieldsOf(t, raw)
	if got["certificate_ref"] != certificate || got["valid_until"] != farDay || got["decision"] != "approved" {
		t.Fatalf("decided send = %v", got)
	}
}
func TestTheCouncilListCarriesTheCertificateAndTheDayItEnds(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("grant = %d %s", status, raw)
	}
	listed := councilFieldsOf(t, r)
	if listed["certificate_ref"] != certificate || listed["valid_until"] != farDay {
		t.Fatalf("council row = %v, want the certificate and the day it ends", listed)
	}
}
func TestTheApplicantReadsTheCertificateAndTheDayItEndsOnReadMine(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("grant = %d %s", status, raw)
	}
	mine := readMineFieldsOf(t, r)
	if mine["certificate_ref"] != certificate || mine["valid_until"] != farDay {
		t.Fatalf("read_mine = %v, want the holder to read the certificate and the day it ends", mine)
	}
}
func TestTheApplicantReadsTheCertificateAndTheDayItEndsOnReadMySend(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("grant = %d %s", status, raw)
	}
	submission := readMySendFieldsOf(t, r)
	if submission["certificate_ref"] != certificate || submission["valid_until"] != farDay {
		t.Fatalf("read_my_send submission = %v, want the holder to read the certificate and the day it ends", submission)
	}
}
func TestAGrantWithOnlyACertificateKeepsItAndNoEndDay(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`"`)); status != http.StatusOK {
		t.Fatalf("grant with a certificate alone = %d %s", status, raw)
	}
	mine := readMineFieldsOf(t, r)
	if mine["certificate_ref"] != certificate {
		t.Fatalf("read_mine = %v, want the certificate", mine)
	}
	if _, has := mine["valid_until"]; has {
		t.Fatalf("read_mine = %v, want no end day", mine)
	}
}
func TestAGrantWithOnlyAnEndDayKeepsItAndNoCertificate(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("grant with an end day alone = %d %s", status, raw)
	}
	mine := readMineFieldsOf(t, r)
	if mine["valid_until"] != farDay {
		t.Fatalf("read_mine = %v, want the end day", mine)
	}
	if _, has := mine["certificate_ref"]; has {
		t.Fatalf("read_mine = %v, want no certificate", mine)
	}
}
func TestAGrantWithoutThemStoresNeitherAndReadsNeither(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody("")); status != http.StatusOK {
		t.Fatalf("plain grant = %d %s", status, raw)
	}
	for name, fields := range map[string]map[string]any{"council list": councilFieldsOf(t, r), "read_mine": readMineFieldsOf(t, r), "read_my_send": readMySendFieldsOf(t, r)} {
		if _, has := fields["certificate_ref"]; has {
			t.Errorf("%s carries certificate_ref: %v", name, fields)
		}
		if _, has := fields["valid_until"]; has {
			t.Errorf("%s carries valid_until: %v", name, fields)
		}
	}
	if fields := councilFieldsOf(t, r); fields["decision"] != "approved" {
		t.Fatalf("council row = %v, want the grant kept", fields)
	}
}
func TestAGrantWithBlankCertificateAndEndDayStoresNeither(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"  ","valid_until":""`)); status != http.StatusOK {
		t.Fatalf("grant with blanks = %d %s", status, raw)
	}
	mine := readMineFieldsOf(t, r)
	if _, has := mine["certificate_ref"]; has {
		t.Fatalf("read_mine = %v, want no certificate", mine)
	}
	if _, has := mine["valid_until"]; has {
		t.Fatalf("read_mine = %v, want no end day", mine)
	}
}
func TestTheCertificateIsTrimmed(t *testing.T) {
	r := sentOnce(t)
	status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"  `+certificate+` \n"`))
	if status != http.StatusOK || fieldsOf(t, raw)["certificate_ref"] != certificate {
		t.Fatalf("grant with a padded certificate = %d %s", status, raw)
	}
}
func TestARefusalCarryingEitherIsRefusedWithBadRequestAndLeavesTheSendUndecided(t *testing.T) {
	for name, extra := range map[string]string{"a certificate": `,"certificate_ref":"` + certificate + `"`, "an end day": `,"valid_until":"` + farDay + `"`, "a certificate and a day": `,"certificate_ref":"` + certificate + `","valid_until":"` + farDay + `"`} {
		r := sentOnce(t)
		status, raw := decideOverTheRoute(t, r, `{"decision":"refused","note":"plan missing"`+extra+`}`)
		if status != http.StatusBadRequest {
			t.Fatalf("refusal carrying %s = %d %s, want 400", name, status, raw)
		}
		if mine := readMineFieldsOf(t, r); mine["decision"] != nil {
			t.Fatalf("send after the refused refusal carrying %s = %v, want undecided", name, mine)
		}
	}
}
func TestARefusalWithoutThemIsStillStored(t *testing.T) {
	r := sentOnce(t)
	status, raw := decideOverTheRoute(t, r, `{"decision":"refused","note":"plan missing"}`)
	if status != http.StatusOK || fieldsOf(t, raw)["decision"] != "refused" {
		t.Fatalf("plain refusal = %d %s", status, raw)
	}
}
func TestAnEndDayInThePastIsRefusedWithBadRequestAndLeavesTheSendUndecided(t *testing.T) {
	r := sentOnce(t)
	status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+pastDay+`"`))
	if status != http.StatusBadRequest {
		t.Fatalf("grant ending in the past = %d %s, want 400", status, raw)
	}
	if mine := readMineFieldsOf(t, r); mine["decision"] != nil {
		t.Fatalf("send after the refused grant = %v, want undecided", mine)
	}
}
func TestAnEndDayThatIsNotADayIsRefusedWithBadRequest(t *testing.T) {
	for _, word := range []string{"31/12/2099", "2099-13-45", "2099-12-31T00:00:00Z", "next year", "2099-1-5"} {
		r := sentOnce(t)
		status, raw := decideOverTheRoute(t, r, grantBody(`,"valid_until":"`+word+`"`))
		if status != http.StatusBadRequest {
			t.Fatalf("grant ending %q = %d %s, want 400", word, status, raw)
		}
		if mine := readMineFieldsOf(t, r); mine["decision"] != nil {
			t.Fatalf("send after the refused grant ending %q = %v, want undecided", word, mine)
		}
	}
}
func TestAnEndDayOfTodayIsAccepted(t *testing.T) {
	r := sentOnce(t)
	today := models.NowRFC3339()[:10]
	status, raw := decideOverTheRoute(t, r, grantBody(`,"valid_until":"`+today+`"`))
	if status != http.StatusOK || fieldsOf(t, raw)["valid_until"] != today {
		t.Fatalf("grant ending today = %d %s, want it accepted", status, raw)
	}
}
func TestDecidingAgainWithoutThemClearsTheEarlierCertificateAndEndDay(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("first grant = %d %s", status, raw)
	}
	if status, raw := decideOverTheRoute(t, r, grantBody("")); status != http.StatusOK {
		t.Fatalf("second grant = %d %s", status, raw)
	}
	mine := readMineFieldsOf(t, r)
	if _, has := mine["certificate_ref"]; has {
		t.Fatalf("read_mine = %v, want the second decision alone", mine)
	}
	if _, has := mine["valid_until"]; has {
		t.Fatalf("read_mine = %v, want the second decision alone", mine)
	}
}
func TestARefusalAfterAGrantClearsTheCertificateAndEndDay(t *testing.T) {
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("grant = %d %s", status, raw)
	}
	if status, raw := decideOverTheRoute(t, r, `{"decision":"refused","note":"withdrawn on review"}`); status != http.StatusOK {
		t.Fatalf("refusal = %d %s", status, raw)
	}
	mine := readMineFieldsOf(t, r)
	if _, has := mine["certificate_ref"]; has {
		t.Fatalf("read_mine = %v, want no certificate after a refusal", mine)
	}
	if _, has := mine["valid_until"]; has {
		t.Fatalf("read_mine = %v, want no end day after a refusal", mine)
	}
}
func TestTheCertificateAndEndDayStayAfterAClerkCorrectsTheSend(t *testing.T) {
	ctx := context.Background()
	r := sentOnce(t)
	if status, raw := decideOverTheRoute(t, r, grantBody(`,"certificate_ref":"`+certificate+`","valid_until":"`+farDay+`"`)); status != http.StatusOK {
		t.Fatalf("grant = %d %s", status, raw)
	}
	if _, err := r.m.CorrectSubmission(ctx, r.form.ID, "mem-1", "clerk-1", []models.Answer{{QuestionID: r.question.ID, ValueText: "street lighting"}}); err != nil {
		t.Fatalf("CorrectSubmission: %v", err)
	}
	mine := readMineFieldsOf(t, r)
	if mine["certificate_ref"] != certificate || mine["valid_until"] != farDay {
		t.Fatalf("read_mine after a correction = %v, want the certificate and the day kept", mine)
	}
}
