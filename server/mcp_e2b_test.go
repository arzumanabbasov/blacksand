package server

import (
	"context"
	"github.com/arzumanabbasov/blacksand/db"
	"strings"
	"testing"
)

func TestE2BMCPDoesNotStartLocalServerWithoutKey(t *testing.T) {
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "")
	client, err := connectMCP(context.Background(), &db.MCPServer{Name: "fixture", Transport: "stdio", Command: "must-never-run-locally"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := client.(*sandboxMCP); !ok {
		t.Fatal("local MCP transport selected")
	}
	_, err = client.Tools(context.Background())
	if err == nil || !strings.Contains(err.Error(), "E2B_API_KEY is required") {
		t.Fatalf("expected missing-key error, got %v", err)
	}
}
