// Package adoption implements Tier-0 adoption contracts: protocol files,
// workflow verification, and audit-based verdict helpers.
package adoption

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RequiredTools are the six MCP tools agents must be instructed to use.
var RequiredTools = []string{
	"mem_context",
	"mem_search",
	"mem_get",
	"mem_record",
	"mem_remember",
	"mem_feedback",
}

// ProtocolMarkers must appear in AGENTS.md and the Cursor rule so agents
// receive the behavioural contract (aligned with docs/mcp-tools.md).
var ProtocolMarkers = []string{
	"mem_context",
	"mem_search",
	"mem_record",
	"mem_remember",
	"mem_feedback",
	"mem_get",
	"## Do not use",
	"outcome=wrong",
	"pending_review",
	"applies_when",
	"invalid_when",
}

// ProtocolFiles relative to repo root.
var ProtocolFiles = []string{
	"AGENTS.md",
	".cursor/rules/rocky-mem-mcp.mdc",
}

// CheckProtocolFiles verifies adoption instruction files exist and contain
// the required tool names and behavioural markers.
func CheckProtocolFiles(repoRoot string) error {
	for _, rel := range ProtocolFiles {
		path := filepath.Join(repoRoot, rel)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("protocol file missing: %s (%w)", rel, err)
		}
		text := string(body)
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("protocol file empty: %s", rel)
		}
		for _, marker := range ProtocolMarkers {
			if !strings.Contains(text, marker) {
				return fmt.Errorf("%s missing required marker %q", rel, marker)
			}
		}
		for _, tool := range RequiredTools {
			if !strings.Contains(text, tool) {
				return fmt.Errorf("%s missing required tool %q", rel, tool)
			}
		}
	}
	// Cursor rule must always apply.
	rulePath := filepath.Join(repoRoot, ".cursor/rules/rocky-mem-mcp.mdc")
	rule, err := os.ReadFile(rulePath)
	if err != nil {
		return err
	}
	if !strings.Contains(string(rule), "alwaysApply: true") {
		return fmt.Errorf("rocky-mem-mcp.mdc must set alwaysApply: true")
	}
	return nil
}

// FindRepoRoot walks upward from start until AGENTS.md is found.
func FindRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("AGENTS.md not found above %s", start)
		}
		dir = parent
	}
}
