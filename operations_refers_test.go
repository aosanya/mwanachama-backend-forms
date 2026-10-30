package mwanachamaforms

import (
	"sort"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

var markedReferences = map[string]string{
	"form_id":     "forms.form",
	"question_id": "forms.question",
}

func TestEveryArgumentNamingAThingAnOperatorWorksWithIsMarkedWithWhatItRefersTo(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	marked := map[string]int{}
	for _, name := range sortedOperationNames(s) {
		for _, a := range s.Operations[name].Args {
			want, isReference := markedReferences[a.As]
			isReference = isReference && !a.Ignored && a.From != "caller" && !a.Whole
			if isReference && a.Refers != want {
				t.Errorf("%s: argument %s refers %q, want %q", name, a.As, a.Refers, want)
			}
			if !isReference && a.Refers != "" {
				t.Errorf("%s: argument %s refers %q but is not one of the things an operator works with", name, a.As, a.Refers)
			}
			if isReference {
				marked[a.As]++
			}
		}
	}
	for as := range markedReferences {
		if marked[as] == 0 {
			t.Errorf("no operation declares %s, so nothing proves it is marked", as)
		}
	}
}

func TestReferencesListsTheMarkedArgumentsAndNoTargetSendOrOtherModuleId(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	references := s.References()
	if len(references) < 45 {
		t.Fatalf("References lists %d arguments, want every form and question id the file declares", len(references))
	}
	seen := map[string]bool{}
	for _, r := range references {
		seen[r.As] = true
		if markedReferences[r.As] != r.Refers {
			t.Errorf("%s.%s is listed as referring %q", r.Operation, r.As, r.Refers)
		}
	}
	var missing []string
	for as := range markedReferences {
		if !seen[as] {
			missing = append(missing, as)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("References never lists %v", missing)
	}
	for _, notAThing := range []string{"target_id", "send_id", "member_id", "actor_id", "key", "period_key", "chapter_id", "respondent_id", "candidate_link_key"} {
		if seen[notAThing] {
			t.Errorf("%s is a row nobody names across modules, another module's id or a key, not a reference", notAThing)
		}
	}
}

func sortedOperationNames(s *dispatch.Spec) []string {
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
