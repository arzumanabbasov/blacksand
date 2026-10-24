package agent

import (
	"context"
	"encoding/json"
	"github.com/Autumn-27/norma/agentcore"
	actool "github.com/Autumn-27/norma/tool"
	"strings"
	"testing"
)

func TestE2BSessionDisablesHostBackgroundInjection(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	opts := executionOptions(agentcore.Options{})
	if !opts.DisableBackgroundTasks {
		t.Fatal("host Monitor could be injected")
	}
	if len(opts.SystemPrompt) != 1 || !strings.Contains(opts.SystemPrompt[0], "E2B Linux sandbox") {
		t.Fatal("sandbox environment missing from guidance")
	}
}

func TestE2BSessionDoesNotInjectLocalWebFetch(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "")
	opts := executionOptions(agentcore.Options{EnableWebFetch: true})
	if opts.EnableWebFetch {
		t.Fatal("SDK could inject a local WebFetch")
	}
	for _, tool := range opts.Tools {
		if tool.Name() != "WebFetch" {
			continue
		}
		result, err := tool.Call(context.Background(), json.RawMessage(`{"url":"https://example.com"}`), &actool.ToolContext{WorkingDir: t.TempDir()})
		if err != nil || !result.IsError || !strings.Contains(result.Flatten(), "E2B_API_KEY is required") {
			t.Fatalf("WebFetch did not fail closed: %+v %v", result, err)
		}
		return
	}
	t.Fatal("remote WebFetch missing")
}
