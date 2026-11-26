package adoption_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/adoption"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := adoption.FindRepoRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckProtocolFiles(t *testing.T) {
	if err := adoption.CheckProtocolFiles(repoRoot(t)); err != nil {
		t.Fatalf("protocol check failed: %v", err)
	}
}

func TestProtocolFilesExist(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range adoption.ProtocolFiles {
		path := filepath.Join(root, rel)
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if st.Size() < 200 {
			t.Fatalf("%s too short (%d bytes)", rel, st.Size())
		}
	}
}

func TestCheckProtocolFilesDetectsMissingMarker(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".cursor/rules"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("mem_context only"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".cursor/rules/rocky-mem-mcp.mdc"), []byte("---\nalwaysApply: true\n---\n"), 0o644)
	err := adoption.CheckProtocolFiles(dir)
	if err == nil {
		t.Fatal("expected failure for incomplete protocol files")
	}
}

func TestRequiredToolsList(t *testing.T) {
	if len(adoption.RequiredTools) != 6 {
		t.Fatalf("want 6 tools, got %d", len(adoption.RequiredTools))
	}
}
