package core_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestIngestPreviewAndConfirm(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		t.Fatal(err)
	}
	svc := core.NewService(st)

	raw := "debug note\napi_key=supersecrettoken123\nkeep this line"
	prev, err := svc.IngestPreview(ctx, core.IngestPreviewInput{Text: raw, Filename: "note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if prev.Hits["generic_secret"] < 1 {
		t.Fatalf("expected secret hit: %+v", prev.Hits)
	}
	if strings.Contains(prev.Redacted, "supersecrettoken123") {
		t.Fatalf("secret leaked in redacted: %s", prev.Redacted)
	}
	rec, err := svc.IngestConfirm(ctx, core.IngestConfirmInput{
		Redacted: prev.Redacted, Filename: "note.md", Title: "Ingest secret sample",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.UID == "" || rec.Trust != "proposed" {
		t.Fatalf("expected proposed experience: %+v", rec)
	}
}

func TestReviewResolveKeepArchivesPeers(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		t.Fatal(err)
	}
	svc := core.NewService(st)
	a, err := svc.Record(ctx, core.RecordInput{
		Title: "Side A", ExpType: "pitfall", Symptom: "a", Fix: "fa",
		AppliesWhen: "x", Outcome: "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "a-evidence"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Record(ctx, core.RecordInput{
		Title: "Side B", ExpType: "pitfall", Symptom: "b", Fix: "fb",
		AppliesWhen: "x", Outcome: "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "b-evidence"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ea, _ := st.GetExperienceByUID(ctx, a.UID)
	eb, _ := st.GetExperienceByUID(ctx, b.UID)
	if err := st.MarkContested(ctx, []int64{ea.ID, eb.ID}, "cg-test"); err != nil {
		t.Fatal(err)
	}
	out, err := svc.ReviewResolve(ctx, a.UID, "keep")
	if err != nil {
		t.Fatal(err)
	}
	if out.Kept != a.UID || len(out.Archived) != 1 || out.Archived[0] != b.UID {
		t.Fatalf("%+v", out)
	}
	ea2, _ := st.GetExperienceByUID(ctx, a.UID)
	eb2, _ := st.GetExperienceByUID(ctx, b.UID)
	if ea2.Trust != "active" || eb2.Trust != "archived" {
		t.Fatalf("trusts %s / %s", ea2.Trust, eb2.Trust)
	}
}
