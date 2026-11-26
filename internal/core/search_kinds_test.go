package core_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestSearchSourceKindReserved(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_, _ = st.EnsureLocalUser(ctx)
	svc := core.NewService(st)

	// Sources are containers; only chunks are indexed, so asking for the
	// container kind is still refused rather than silently empty.
	_, err = svc.Search(ctx, core.SearchInput{Query: "x", Kinds: []string{"source"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.As(err).Code != errs.NotImplemented {
		t.Fatalf("got %v", err)
	}
}

func TestIngestThenSearchChunkLandsInBackground(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_, _ = st.EnsureLocalUser(ctx)
	svc := core.NewService(st)

	body := "# Rollout\n\nThe checksum annotation forces pod recreation after a config change.\n\n## Probes\n\nlivenessProbe initialDelay must exceed startup time."
	out, err := svc.IngestConfirm(ctx, core.IngestConfirmInput{
		Redacted: body, Filename: "rollout.md", Format: "plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ChunkCount < 2 {
		t.Fatalf("expected heading-split chunks, got %d", out.ChunkCount)
	}

	pack, err := svc.Search(ctx, core.SearchInput{Query: "checksum annotation", Cwd: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Pack, "## Background") {
		t.Fatalf("corpus not in background: %s", pack.Pack)
	}
	if !strings.Contains(pack.Pack, "c_") {
		t.Fatalf("missing chunk uid: %s", pack.Pack)
	}

	// An explicit chunk request allows more, still capped per source.
	explicit, err := svc.Search(ctx, core.SearchInput{Query: "probes", Kinds: []string{"chunk"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(explicit.Pack, "## Background") {
		t.Fatalf("explicit chunk search: %s", explicit.Pack)
	}
}
