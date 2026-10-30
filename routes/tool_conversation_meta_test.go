package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamaforms "github.com/aosanya/mwanachama-backend-forms"
	"github.com/aosanya/mwanachama-backend-forms/models"
	"github.com/aosanya/mwanachama-backend-forms/routes"
)

type conversationRecorder struct {
	mwanachamaforms.FormManager
	mu   sync.Mutex
	seen []string
}

func (r *conversationRecorder) GetIn(ctx context.Context, formID, language string) (models.Form, error) {
	r.mu.Lock()
	r.seen = append(r.seen, dispatch.ConversationOf(ctx))
	r.mu.Unlock()
	return r.FormManager.GetIn(ctx, formID, language)
}

func readFormTool(t *testing.T, recorder *conversationRecorder, mount routes.Mount) routes.Tool {
	t.Helper()
	tools, err := routes.BuildGatedTools(recorder, mount)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if tool.Action == "forms.form.read" {
			return tool
		}
	}
	t.Fatal("no forms.form.read tool")
	return routes.Tool{}
}

func conversationsSeenBy(t *testing.T, mount routes.Mount, meta map[string]any) ([]string, error) {
	t.Helper()
	recorder := &conversationRecorder{FormManager: newTestManager(t)}
	args, _ := json.Marshal(map[string]any{"form_id": "any"})
	_, err := readFormTool(t, recorder, mount).Handle(context.Background(), dispatch.Request{Arguments: args, Meta: meta})
	return recorder.seen, err
}

func TestAToolCallCarryingMetaConversationIdReachesTheManagerOnItsContext(t *testing.T) {
	for name, mount := range map[string]routes.Mount{"plain": {}, "reached only": {ReachedOnly: true}} {
		seen, _ := conversationsSeenBy(t, mount, map[string]any{dispatch.ConversationMetaKey: "chat-42"})
		if len(seen) != 1 || seen[0] != "chat-42" {
			t.Fatalf("%s: the manager saw conversations %q, want exactly [chat-42]", name, seen)
		}
	}
}

func TestAToolCallCarryingNoMetaReachesTheManagerWithNoConversation(t *testing.T) {
	seen, _ := conversationsSeenBy(t, routes.Mount{}, nil)
	if len(seen) != 1 || seen[0] != "" {
		t.Fatalf("the manager saw conversations %q, want exactly an empty one", seen)
	}
}

func TestAToolCallCarryingAMalformedConversationIdNeverReachesTheManager(t *testing.T) {
	seen, err := conversationsSeenBy(t, routes.Mount{}, map[string]any{dispatch.ConversationMetaKey: "not a conversation"})
	if !errors.Is(err, dispatch.ErrInvalidConversation) {
		t.Fatalf("want ErrInvalidConversation, got %v", err)
	}
	if len(seen) != 0 {
		t.Fatalf("the manager was called %d times", len(seen))
	}
}
