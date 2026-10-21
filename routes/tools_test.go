package routes

import (
	"slices"
	"testing"

	forms "github.com/aosanya/mwanachama-backend-forms"
)

func TestGatedToolsLeaveTheAnonymousActionsOut(t *testing.T) {
	tools, err := BuildGatedTools(shapeOnly{}, Mount{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) == 0 {
		t.Fatal("no tools built")
	}
	for _, tool := range tools {
		if slices.Contains(AnonymousActions, tool.Action) {
			t.Fatalf("anonymous action %s became a tool", tool.Action)
		}
		if tool.Name == "" || len(tool.InputSchema) == 0 {
			t.Fatalf("tool for %s has no name or schema", tool.Action)
		}
	}
}

type shapeOnly struct{ forms.FormManager }
