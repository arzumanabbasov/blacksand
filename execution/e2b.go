// Package execution routes agent-created commands and workspace files to E2B.
package execution

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	actool "github.com/Autumn-27/norma/tool"
)

// Enabled deliberately does not accept unknown providers as a local fallback.
func Enabled() bool {
	c := CurrentConfig()
	return strings.TrimSpace(c.Mode) != "" && c.Mode != "local"
}

type workspace struct {
	sync.Mutex
	id string
}

var workspaces sync.Map

type response struct {
	SandboxID string `json:"sandboxId"`
	Text      string `json:"text"`
	Error     string `json:"error"`
}

// Call launches only the fixed SDK bridge locally. Agent input is sent on stdin,
// never interpolated into a local shell command. No host environment is copied
// into the sandbox; provider credentials remain on the control plane.
func Call(ctx context.Context, name string, input json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
	c := CurrentConfig()
	if c.Mode != "e2b" {
		return actool.Errorf("Unknown execution provider; use local or e2b"), nil
	}
	if c.APIKey == "" {
		return actool.Errorf("E2B_API_KEY is required; local execution is disabled"), nil
	}
	if tc == nil || tc.WorkingDir == "" {
		return actool.Errorf("E2B execution requires a workspace"), nil
	}
	key, err := filepath.Abs(tc.WorkingDir)
	if err != nil {
		return actool.Errorf(err.Error()), nil
	}
	v, _ := workspaces.LoadOrStore(key, &workspace{})
	w := v.(*workspace)
	// Serialize operations in a workspace, including sandbox creation and edits.
	w.Lock()
	defer w.Unlock()
	if ctx.Err() != nil {
		return actool.Errorf("E2B call cancelled before execution"), nil
	}
	digest := sha256.Sum256([]byte(key))
	bridge := os.Getenv("BLACKSAND_E2B_BRIDGE")
	if bridge == "" {
		bridge = filepath.Join("adapters", "e2b", "bridge.mjs")
	}
	bridge, err = filepath.Abs(bridge)
	if err != nil {
		return actool.Errorf(err.Error()), nil
	}
	cancelFile, err := os.CreateTemp("", "blacksand-e2b-cancel-*")
	if err != nil {
		return actool.Errorf("Cannot create E2B cancellation marker"), nil
	}
	cancelFile.Close()
	defer os.Remove(cancelFile.Name())
	body, err := json.Marshal(map[string]any{"name": name, "input": input, "sandboxId": w.id, "workspace": hex.EncodeToString(digest[:])[:24], "hostWorkDir": key, "cancelPath": cancelFile.Name()})
	if err != nil {
		return actool.Errorf(err.Error()), nil
	}
	cmd := exec.Command("node", bridge)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "E2B_API_KEY=") && !strings.HasPrefix(entry, "BLACKSAND_E2B_TEMPLATE=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "E2B_API_KEY="+c.APIKey, "BLACKSAND_E2B_TEMPLATE="+c.Template)
	cmd.Stdin = strings.NewReader(string(body))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return actool.Errorf("E2B bridge unavailable"), nil
	}
	// SDK errors are sanitized by the bridge. Do not expose Node stderr/env.
	if err := cmd.Start(); err != nil {
		return actool.Errorf("Cannot start E2B bridge: install Node and adapter dependencies"), nil
	}
	done := make(chan struct{})
	watcherDone := make(chan struct{})
	defer func() { close(done); <-watcherDone }()
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			_ = os.WriteFile(cancelFile.Name(), []byte("cancel"), 0600)
		case <-done:
		}
	}()
	var final response
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	for scanner.Scan() {
		var r response
		if json.Unmarshal(scanner.Bytes(), &r) != nil {
			continue
		}
		if r.SandboxID != "" {
			w.id = r.SandboxID
		}
		final = r
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		return actool.Errorf("E2B call cancelled; inspect sandbox state before retrying"), nil
	}
	if scanner.Err() != nil || err != nil {
		return actool.Errorf("E2B bridge failed; local execution is disabled"), nil
	}
	if final.Error != "" {
		return actool.Errorf(final.Error), nil
	}
	return actool.Text(final.Text), nil
}

type remoteTool struct{ actool.CoreTool }

func (t *remoteTool) Description() string {
	return t.CoreTool.Description() + " Execution and files are in an E2B Linux sandbox; use relative paths. No local fallback."
}
func (t *remoteTool) Prompt() string {
	return "Use relative paths in the E2B workspace. Foreground commands only. Interactive shell and background task tools are unavailable."
}
func (t *remoteTool) Call(ctx context.Context, in json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
	return Call(ctx, t.Name(), in, tc)
}

type blockedTool struct{ actool.CoreTool }

func (t *blockedTool) Call(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
	return actool.Errorf(fmt.Sprintf("%s is unavailable in E2B execution mode; use foreground Bash", t.Name())), nil
}

// Wrap is applied before catalog/default decoration, preserving permission checks
// while replacing handlers that would execute or access the host workspace.
func Wrap(tools []actool.CoreTool) []actool.CoreTool {
	if !Enabled() {
		return tools
	}
	out := append([]actool.CoreTool(nil), tools...)
	for i, t := range out {
		switch t.Name() {
		case "Bash", "Read", "Write", "Edit", "MultiEdit", "LS", "Glob", "Grep", "WebFetch":
			out[i] = &remoteTool{t}
		case "TaskOutput", "TaskStop", "TaskList", "Monitor", "shell_open", "shell_read", "shell_write", "shell_resize", "shell_close":
			out[i] = &blockedTool{t}
		}
	}
	return out
}

// BlockUnsupported also covers shell tools injected by the catalog resolver.
func BlockUnsupported(tools []actool.CoreTool) []actool.CoreTool {
	if !Enabled() {
		return tools
	}
	for i, t := range tools {
		if strings.HasPrefix(t.Name(), "shell_") || t.Name() == "TaskOutput" || t.Name() == "TaskStop" || t.Name() == "TaskList" || t.Name() == "Monitor" {
			tools[i] = &blockedTool{t}
		}
	}
	return tools
}
