package mwanachamaforms_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func TestF41_TheWaitingAndAnsweredListsServeReviewedTitles(t *testing.T) {
	mgr, formID, qID := publishedToTwoGroups(t)
	ctx := context.Background()
	put := func(lang, text string, reviewed bool) {
		t.Helper()
		if _, err := mgr.PutTranslation(ctx, models.Translation{
			FormID: formID, SubjectRole: models.TranslationOfForm, SubjectID: formID,
			Field: "title", Language: lang, Text: text, Reviewed: reviewed,
		}); err != nil {
			t.Fatalf("PutTranslation: %v", err)
		}
	}
	put("sw", "Mzunguko", true)
	put("fr", "Tour", false)
	for _, c := range []struct{ lang, want string }{{"sw", "Mzunguko"}, {"fr", "Round"}, {"de", "Round"}, {"", "Round"}} {
		waiting, err := mgr.ListWaitingIn(ctx, "alice", c.lang)
		if err != nil || len(waiting) != 1 || waiting[0].Title != c.want {
			t.Fatalf("ListWaitingIn(%q) = %+v, %v; want title %q", c.lang, waiting, err, c.want)
		}
	}
	for _, g := range []string{"traders", "teachers"} {
		if _, err := mgr.SubmitAnswersAs(ctx, formID, "alice", g, []models.Answer{{QuestionID: qID, ValueText: "ok"}}); err != nil {
			t.Fatalf("SubmitAnswersAs %s: %v", g, err)
		}
	}
	for _, c := range []struct{ lang, want string }{{"sw", "Mzunguko"}, {"fr", "Round"}, {"", "Round"}} {
		answered, err := mgr.ListAnsweredIn(ctx, "alice", c.lang)
		if err != nil || len(answered) != 2 {
			t.Fatalf("ListAnsweredIn(%q) = %+v, %v; want two rows", c.lang, answered, err)
		}
		for _, row := range answered {
			if row.Title != c.want {
				t.Fatalf("ListAnsweredIn(%q) row %+v; want title %q", c.lang, row, c.want)
			}
		}
	}
}
