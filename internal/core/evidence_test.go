package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEvidencePath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	body, warn := resolveEvidenceText("path", "notes.md", "", dir)
	if warn != "" {
		t.Fatalf("warn %s", warn)
	}
	if !strings.Contains(body, "path notes.md") || !strings.Contains(body, "mtime=") {
		t.Fatalf("body %q", body)
	}
}

func TestResolveEvidenceCommitFallback(t *testing.T) {
	body, warn := resolveEvidenceText("commit", "deadbeef", "", "/no/such/repo")
	if body != "commit deadbeef" {
		t.Fatalf("body %q", body)
	}
	if warn == "" {
		t.Fatal("expected warning")
	}
}
