//go:build ignore

// MCP stdio smoke test: spawn rockymem mcp and call all six tools.
//
//	go run ./scripts/mcp_smoke.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	bin := filepath.Join(root, "bin", "rockymem")
	db := filepath.Join(root, ".local", "rocky-mem.db")
	if _, err := os.Stat(bin); err != nil {
		fatal(fmt.Errorf("build first: make build (%w)", err))
	}
	if _, err := os.Stat(db); err != nil {
		fatal(fmt.Errorf("seed first: ./scripts/seed-demo.sh (%w)", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "--dsn", "sqlite://"+db, "mcp")
	client := mcp.NewClient(&mcp.Implementation{Name: "rocky-mem-smoke", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		fatal(err)
	}
	fmt.Println("tools:")
	want := map[string]bool{
		"mem_context": false, "mem_search": false, "mem_get": false,
		"mem_record": false, "mem_remember": false, "mem_feedback": false,
	}
	for _, t := range tools.Tools {
		fmt.Printf("  - %s\n", t.Name)
		if _, ok := want[t.Name]; ok {
			want[t.Name] = true
		}
	}
	for name, ok := range want {
		if !ok {
			fatal(fmt.Errorf("missing tool %s", name))
		}
	}

	cwd := root
	mustCall(ctx, session, "mem_context", map[string]any{"cwd": cwd})
	search := mustCall(ctx, session, "mem_search", map[string]any{
		"query": "ConfigMap rollout", "cwd": cwd,
	})
	var pack struct {
		UIDs []string `json:"uids"`
		Pack string   `json:"pack"`
	}
	if err := json.Unmarshal([]byte(search), &pack); err != nil {
		fatal(err)
	}
	if len(pack.UIDs) == 0 {
		fatal(fmt.Errorf("mem_search returned no uids"))
	}
	fmt.Printf("mem_search uids=%v\n", pack.UIDs)

	expUID := ""
	for _, u := range pack.UIDs {
		if len(u) > 2 && u[:2] == "e_" {
			expUID = u
			break
		}
	}
	if expUID == "" {
		fatal(fmt.Errorf("no experience uid in search result"))
	}
	mustCall(ctx, session, "mem_get", map[string]any{"uid": expUID, "include_raw": true})
	mustCall(ctx, session, "mem_feedback", map[string]any{
		"uid": expUID, "outcome": "helpful",
	})
	mustCall(ctx, session, "mem_remember", map[string]any{
		"title": "mcp smoke", "body": "MCP smoke test wrote this memory.",
		"mem_key": "mcp_smoke", "mem_type": "fact", "scope": "session", "cwd": cwd,
	})
	mustCall(ctx, session, "mem_record", map[string]any{
		"title": "MCP smoke experience", "exp_type": "howto",
		"symptom": "need to verify MCP stdio works",
		"fix": "run scripts/mcp_smoke.go",
		"applies_when": "after changing MCP tool surface",
		"invalid_when": "when only testing CLI",
		"outcome": "success", "confidence": 0.9, "cwd": cwd,
		"evidence": []map[string]any{
			{"type": "quote", "text": "smoke test called mem_record over MCP stdio"},
		},
	})

	fmt.Println("OK: all six MCP tools responded over stdio")
}

func mustCall(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) string {
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		fatal(fmt.Errorf("%s: %w", name, err))
	}
	if res.IsError {
		fatal(fmt.Errorf("%s returned isError: %s", name, textOf(res)))
	}
	out := textOf(res)
	fmt.Printf("%s: ok (%d bytes)\n", name, len(out))
	return out
}

func textOf(res *mcp.CallToolResult) string {
	var b string
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			b += t.Text
		}
	}
	return b
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "FAIL:", err)
	os.Exit(1)
}
