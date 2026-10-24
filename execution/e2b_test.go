package execution

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	actool "github.com/Autumn-27/norma/tool"
)

func TestE2BMissingKeyNeverExecutesHostCommand(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "")
	tools := Wrap([]actool.CoreTool{actool.NewBash()})
	result, err := tools[0].Call(context.Background(), json.RawMessage(`{"command":"echo must-not-run"}`), &actool.ToolContext{WorkingDir: t.TempDir()})
	if err != nil || !result.IsError || result.Flatten() != "E2B_API_KEY is required; local execution is disabled" {
		t.Fatalf("unexpected result: %+v, %v", result, err)
	}
}

func TestUnknownProviderFailsClosed(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "typo")
	result, _ := Call(context.Background(), "Bash", nil, nil)
	if !result.IsError {
		t.Fatal("unknown provider accepted")
	}
}

func TestE2BWebFetchUsesRemoteHandler(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "")
	original := actool.NewWebFetch(actool.WebFetchConfig{})
	wrapped := Wrap([]actool.CoreTool{original})[0]
	if wrapped == original {
		t.Fatal("WebFetch remained local")
	}
	result, err := wrapped.Call(context.Background(), json.RawMessage(`{"url":"https://example.com"}`), &actool.ToolContext{WorkingDir: t.TempDir()})
	if err != nil || !result.IsError || !strings.Contains(result.Flatten(), "E2B_API_KEY is required") {
		t.Fatalf("unexpected result: %+v %v", result, err)
	}
}

func TestLocalModePreservesTool(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "local")
	bash := actool.NewBash()
	if Wrap([]actool.CoreTool{bash})[0] != bash {
		t.Fatal("local handler changed")
	}
}

func TestE2BBridgeReusesSandboxAndTransportsCodeAsData(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node unavailable")
	}
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "test-key-not-used-by-a-provider")
	bridge := filepath.Join(t.TempDir(), "bridge.mjs")
	code := `let raw=''; for await (const part of process.stdin) raw+=part; const r=JSON.parse(raw); console.log(JSON.stringify({sandboxId:'fixture-sandbox'})); console.log(JSON.stringify({text:JSON.stringify({previous:r.sandboxId,input:r.input})}));`
	if err := os.WriteFile(bridge, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BLACKSAND_E2B_BRIDGE", bridge)
	tc := &actool.ToolContext{WorkingDir: t.TempDir()}
	input := json.RawMessage(`{"command":"$(must-never-execute-on-host); python3 script.py"}`)
	first, err := Call(context.Background(), "Bash", input, tc)
	if err != nil || first.IsError || !strings.Contains(first.Flatten(), "must-never-execute-on-host") {
		t.Fatalf("request failed: %+v %v", first, err)
	}
	second, err := Call(context.Background(), "Bash", input, tc)
	if err != nil || second.IsError || !strings.Contains(second.Flatten(), `"previous":"fixture-sandbox"`) {
		t.Fatalf("sandbox not reused: %+v %v", second, err)
	}
}

func TestE2BCancellationReachesBridge(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node unavailable")
	}
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "fixture")
	bridge := filepath.Join(t.TempDir(), "bridge.mjs")
	code := `import {statSync} from 'node:fs'; let raw=''; for await (const part of process.stdin) raw+=part; const r=JSON.parse(raw); const timer=setInterval(()=>{if(statSync(r.cancelPath).size){clearInterval(timer); console.log(JSON.stringify({error:'cancelled'}));}},20);`
	if err := os.WriteFile(bridge, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BLACKSAND_E2B_BRIDGE", bridge)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	result, err := Call(ctx, "Bash", json.RawMessage(`{"command":"never executed"}`), &actool.ToolContext{WorkingDir: t.TempDir()})
	if err != nil || !result.IsError || !strings.Contains(result.Flatten(), "cancelled") {
		t.Fatalf("cancel not delivered: %+v %v", result, err)
	}
}
