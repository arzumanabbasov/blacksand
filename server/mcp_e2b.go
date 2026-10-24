package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
	"github.com/arzumanabbasov/blacksand/db"
	"github.com/arzumanabbasov/blacksand/execution"
)

type sandboxMCP struct {
	server  *db.MCPServer
	workdir string
}

func (c *sandboxMCP) Close() error { return nil }
func (c *sandboxMCP) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c.server.Insecure {
		return nil, fmt.Errorf("E2B MCP requires verified TLS; disable insecure mode")
	}
	args, env := jsonStrSlice(c.server.Args), jsonStrMap(c.server.Env)
	if c.server.Name == "browser" {
		args = stripProxyArgs(args)
		delete(env, "NODE_EXTRA_CA_CERTS")
	}
	body, _ := json.Marshal(map[string]any{"timeout_ms": 300000, "method": method, "params": params, "server": map[string]any{
		"transport": c.server.Transport, "command": c.server.Command, "args": args, "env": env, "url": c.server.URL,
	}})
	result, err := execution.Call(ctx, "MCP", body, &actool.ToolContext{WorkingDir: c.workdir})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf("%s", result.Flatten())
	}
	raw := json.RawMessage(result.Flatten())
	if !json.Valid(raw) {
		return nil, fmt.Errorf("Invalid remote MCP response")
	}
	return raw, nil
}
func (c *sandboxMCP) Tools(ctx context.Context) ([]actool.CoreTool, error) {
	raw, err := c.request(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var response struct {
		Tools []struct {
			Name        string
			Description string
			InputSchema map[string]any
		}
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	var tools []actool.CoreTool
	for _, item := range response.Tools {
		item := item
		name := "mcp__" + c.server.Name + "__" + item.Name
		tools = append(tools, actool.Build(actool.Spec{Name: name, Description: item.Description, Schema: item.InputSchema,
			Permissions: func(context.Context, json.RawMessage, permission.Context) permission.Decision {
				return permission.AskUser("call MCP tool " + name + "?")
			},
			Run: func(ctx context.Context, in json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
				raw, err := c.request(ctx, "tools/call", map[string]any{"name": item.Name, "arguments": in})
				if err != nil {
					return actool.Errorf(err.Error()), nil
				}
				var response struct {
					Content []struct {
						Type string
						Text string
					}
					IsError bool
				}
				if err := json.Unmarshal(raw, &response); err != nil {
					return actool.Errorf("Invalid MCP response"), nil
				}
				text := ""
				for _, block := range response.Content {
					if block.Type == "text" {
						text += block.Text + "\n"
					} else {
						text += "[Non-text MCP content omitted]\n"
					}
				}
				result := actool.Text(text)
				result.IsError = response.IsError
				return result, nil
			},
		}))
	}
	return tools, nil
}
func newSandboxMCP(m *db.MCPServer) *sandboxMCP {
	// A backend-owned remote workspace keeps discovery and calls on the same MCP
	// connection. Nothing is read from or written to this local directory.
	root, _ := os.Getwd()
	return &sandboxMCP{server: m, workdir: filepath.Join(root, ".e2b-mcp", fmt.Sprint(m.ID))}
}
