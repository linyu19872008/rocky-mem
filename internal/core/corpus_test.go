package core_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/store/sqlite"
)

func openSvc(t *testing.T) (*sqlite.DB, *core.Service) {
	t.Helper()
	st, err := sqlite.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		t.Fatal(err)
	}
	return st, core.NewService(st)
}

func TestIngestSameContentIsIdempotent(t *testing.T) {
	st, svc := openSvc(t)
	ctx := context.Background()
	in := core.IngestConfirmInput{
		Redacted: "The checksum annotation forces pod recreation.\n\nSecond paragraph.",
		Filename: "rollout.md", Format: "plain",
	}
	first, err := svc.IngestConfirm(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.IngestConfirm(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.SourceUID == "" || first.Unchanged {
		t.Fatalf("first ingest: %+v", first)
	}
	if !second.Unchanged || second.SourceUID != first.SourceUID {
		t.Fatalf("second ingest should reuse source: %+v vs %+v", second, first)
	}
	stats, err := st.CorpusStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sources != 1 || stats.Chunks != first.ChunkCount {
		t.Fatalf("stats after double ingest: %+v", stats)
	}
}

func TestIngestReingestArchivesPrevious(t *testing.T) {
	st, svc := openSvc(t)
	ctx := context.Background()

	v1, err := svc.IngestConfirm(ctx, core.IngestConfirmInput{
		Redacted: "rollout notes version one mentions an obsolete flag",
		Filename: "design.md", Format: "plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	v2, err := svc.IngestConfirm(ctx, core.IngestConfirmInput{
		Redacted: "rollout notes version two mentions the checksum annotation",
		Filename: "design.md", Format: "plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	if v2.ArchivedPrevious != 1 {
		t.Fatalf("expected previous source archived, got %d", v2.ArchivedPrevious)
	}

	old, err := st.GetSourceByUID(ctx, v1.SourceUID)
	if err != nil {
		t.Fatal(err)
	}
	if old == nil || old.Status != "archived" || old.SupersededBy == nil {
		t.Fatalf("old source not archived: %+v", old)
	}

	// The archived generation must not come back through retrieval.
	pack, err := svc.Search(ctx, core.SearchInput{Query: "obsolete flag"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pack.Pack, "obsolete flag") {
		t.Fatalf("archived generation still searchable: %s", pack.Pack)
	}
	pack, err = svc.Search(ctx, core.SearchInput{Query: "checksum annotation"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Pack, "checksum annotation") {
		t.Fatalf("new generation missing: %s", pack.Pack)
	}
}

func TestIngestCorpusIsNotAnEntry(t *testing.T) {
	st, svc := openSvc(t)
	ctx := context.Background()
	out, err := svc.IngestConfirm(ctx, core.IngestConfirmInput{
		Redacted: "just some background material", Filename: "notes.md", Format: "plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.UID != "" {
		t.Fatalf("corpus mode must not create an experience: %+v", out)
	}
	var experiences, memories int
	if err := st.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	// Doctor reports zero entries but a non-empty corpus.
	rep, err := st.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	experiences, memories = rep.ExperienceCount, rep.MemoryCount
	if experiences != 0 || memories != 0 {
		t.Fatalf("expected no entries, got exp=%d mem=%d", experiences, memories)
	}
	if rep.SourceCount != 1 || rep.ActiveChunks != out.ChunkCount || !rep.SearchDocOK {
		t.Fatalf("doctor corpus: %+v", rep)
	}
}
