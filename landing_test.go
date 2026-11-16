package mwanachamaforms_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

type collectingWriter struct {
	mu      sync.Mutex
	written []models.LandedRecord
	failure error
}

func (w *collectingWriter) Write(_ context.Context, r models.LandedRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.failure != nil {
		return w.failure
	}
	w.written = append(w.written, r)
	return nil
}

func writersFor(module string, w forms.RecordWriter) forms.RecordWriters {
	return func(asked string) (forms.RecordWriter, bool) {
		if asked == module {
			return w, true
		}
		return nil, false
	}
}

func personLandsIn() models.LandsIn {
	return models.LandsIn{
		Record:  "catalog.person",
		KeyedBy: []string{"member", "full_name"},
		FieldMap: map[string]string{
			"Full name":   "full_name",
			"Ward":        "ward",
			"Has a phone": "has_phone",
		},
	}
}

type roundOf struct {
	m    forms.FormManager
	form models.Form
	name models.Question
	ward models.Question
}

func openableRound(t *testing.T, m forms.FormManager, landsIn models.LandsIn, extra ...string) roundOf {
	t.Helper()
	ctx := context.Background()
	f, err := m.Create(ctx, models.Form{
		Title: "Synthetic registration", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf,
		LandsIn: landsIn,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ask := func(prompt string) models.Question {
		q, _, err := m.AddQuestion(ctx, models.QuestionDraft{Question: models.Question{FormID: f.ID, Prompt: prompt, AnswerType: models.AnswerFreeText}})
		if err != nil {
			t.Fatalf("AddQuestion %q: %v", prompt, err)
		}
		return q
	}
	r := roundOf{m: m, form: f, name: ask("Full name"), ward: ask("Ward")}
	for _, prompt := range extra {
		ask(prompt)
	}
	return r
}

func (r roundOf) send(t *testing.T, member, name, ward string) error {
	t.Helper()
	_, err := r.m.SubmitAnswers(context.Background(), r.form.ID, member, []models.Answer{
		{QuestionID: r.name.ID, ValueText: name},
		{QuestionID: r.ward.ID, ValueText: ward},
	})
	return err
}

func TestARoundWithAnUnmappedQuestionCannotBeOpened(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", &collectingWriter{}))
	r := openableRound(t, m, personLandsIn(), "Phone number", "Street")

	_, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", "")
	if !errors.Is(err, forms.ErrUnmappedQuestions) {
		t.Fatalf("Publish = %v, want ErrUnmappedQuestions", err)
	}
	for _, prompt := range []string{"Phone number", "Street"} {
		if !strings.Contains(err.Error(), prompt) {
			t.Fatalf("error %q does not name %q", err, prompt)
		}
	}
	if strings.Contains(err.Error(), "Full name") {
		t.Fatalf("error %q names a mapped question", err)
	}
	got, _ := m.Get(context.Background(), r.form.ID)
	if got.Status != models.StatusDraft {
		t.Fatalf("status = %s, want draft", got.Status)
	}
}

func TestARoundWhoseQuestionsAreAllMappedOpens(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", &collectingWriter{}))
	r := openableRound(t, m, personLandsIn(), "has a phone")
	opened, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", "")
	if err != nil {
		t.Fatalf("Publish = %v", err)
	}
	if opened.Status != models.StatusOpen {
		t.Fatalf("status = %s, want open", opened.Status)
	}
	if opened.LandsIn.Record != "catalog.person" || opened.LandsIn.FieldMap["Ward"] != "ward" {
		t.Fatalf("lands_in did not survive storage: %+v", opened.LandsIn)
	}
}

func TestARoundLandingInAModuleWithNoWriterCannotBeOpened(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", &collectingWriter{}))
	landsIn := personLandsIn()
	landsIn.Record = "accounting.invoice"
	r := openableRound(t, m, landsIn, "has a phone")
	_, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", "")
	if !errors.Is(err, forms.ErrNoRecordWriter) || !strings.Contains(err.Error(), "accounting") {
		t.Fatalf("Publish = %v, want ErrNoRecordWriter naming accounting", err)
	}
}

func TestARoundWithNoWritersWiredCannotBeOpened(t *testing.T) {
	m := gatedManager(t, nil)
	r := openableRound(t, m, personLandsIn(), "has a phone")
	if _, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", ""); !errors.Is(err, forms.ErrNoRecordWriter) {
		t.Fatalf("Publish = %v, want ErrNoRecordWriter", err)
	}
}

func TestAFormWithoutLandsInOpensWithNoWriters(t *testing.T) {
	m := gatedManager(t, nil)
	r := openableRound(t, m, models.LandsIn{}, "Anything")
	if _, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish = %v", err)
	}
}

func TestALandsInNotNamedForAModuleIsRefusedWhenComposed(t *testing.T) {
	m := gatedManager(t, nil)
	landsIn := personLandsIn()
	landsIn.Record = "person"
	_, err := m.Create(context.Background(), models.Form{
		Title: "Synthetic", ClosesAt: "2099-01-01T00:00:00Z", OriginatorChapterID: "g-1",
		CreatedBy: "author-1", Audience: models.AudienceMember, CollectionMode: models.CollectionSelf, LandsIn: landsIn,
	})
	if !errors.Is(err, forms.ErrInvalidForm) {
		t.Fatalf("Create = %v, want ErrInvalidForm", err)
	}
}

func TestLandsInIsEditableOnADraftAndRefusedWhenMalformed(t *testing.T) {
	m := gatedManager(t, nil)
	ctx := context.Background()
	r := openableRound(t, m, models.LandsIn{})
	edit := r.form
	edit.LandsIn = personLandsIn()
	updated, err := m.Update(ctx, edit)
	if err != nil || updated.LandsIn.Record != "catalog.person" {
		t.Fatalf("Update = %+v, %v", updated.LandsIn, err)
	}
	edit.LandsIn.FieldMap = nil
	if _, err := m.Update(ctx, edit); !errors.Is(err, forms.ErrInvalidForm) {
		t.Fatalf("Update with no field_map = %v, want ErrInvalidForm", err)
	}
}

func TestASendCreatesAndAResendUpdatesTheRecord(t *testing.T) {
	w := &collectingWriter{}
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", w))
	r := openableRound(t, m, personLandsIn())
	if _, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := r.send(t, "mem-1", "Ilona Test", "North"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := r.send(t, "mem-1", "Ilona Test", "South"); err != nil {
		t.Fatalf("resend: %v", err)
	}
	if len(w.written) != 2 {
		t.Fatalf("records written = %d, want 2", len(w.written))
	}
	first, second := w.written[0], w.written[1]
	if first.Record != "catalog.person" || first.Fields["full_name"] != "Ilona Test" || first.Fields["ward"] != "North" {
		t.Fatalf("first = %+v", first)
	}
	if first.Key["member"] != "mem-1" || first.Key["full_name"] != "Ilona Test" {
		t.Fatalf("key = %+v", first.Key)
	}
	if second.Fields["ward"] != "South" || second.Key["member"] != first.Key["member"] || second.Key["full_name"] != first.Key["full_name"] {
		t.Fatalf("a resend must carry the same key with the new value: first %+v second %+v", first, second)
	}
}

func TestAFailedWriteIsReportedAndTheAnswersAreKept(t *testing.T) {
	w := &collectingWriter{failure: errors.New("the register is down")}
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", w))
	r := openableRound(t, m, personLandsIn())
	if _, _, err := m.Publish(context.Background(), r.form.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	err := r.send(t, "mem-1", "Ilona Test", "North")
	if !errors.Is(err, forms.ErrLandingFailed) {
		t.Fatalf("send = %v, want ErrLandingFailed", err)
	}
	kept, _ := m.ListAnswers(context.Background(), r.form.ID, "mem-1")
	if len(kept) != 2 {
		t.Fatalf("kept answers = %d, want 2", len(kept))
	}
}

func TestATestRunValidatesTheRecordButNeverWritesIt(t *testing.T) {
	live := &collectingWriter{}
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", live))
	run, err := m.ForRun("run-1")
	if err != nil {
		t.Fatalf("ForRun: %v", err)
	}
	r := openableRound(t, run, personLandsIn())
	if _, _, err := run.Publish(context.Background(), r.form.ID, "", "author-1", ""); err != nil {
		t.Fatalf("Publish in a run: %v", err)
	}
	if err := r.send(t, "mem-1", "Ilona Test", "North"); err != nil {
		t.Fatalf("send in a run: %v", err)
	}
	if len(live.written) != 0 {
		t.Fatalf("a test run wrote %d live records", len(live.written))
	}
}

func TestATestRunStillRefusesAModuleWithNoWriter(t *testing.T) {
	m := gatedManager(t, nil)
	m.UseRecordWriters(writersFor("catalog", &collectingWriter{}))
	run, _ := m.ForRun("run-1")
	landsIn := personLandsIn()
	landsIn.Record = "accounting.invoice"
	r := openableRound(t, run, landsIn)
	if _, _, err := run.Publish(context.Background(), r.form.ID, "", "author-1", ""); !errors.Is(err, forms.ErrNoRecordWriter) {
		t.Fatalf("Publish in a run = %v, want ErrNoRecordWriter", err)
	}
}
