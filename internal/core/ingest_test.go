package core_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
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
	if prev.Format != "plain" {
		t.Fatalf("format=%s", prev.Format)
	}
	if prev.Hits["generic_secret"] < 1 {
		t.Fatalf("expected secret hit: %+v", prev.Hits)
	}
	if strings.Contains(prev.Redacted, "supersecrettoken123") {
		t.Fatalf("secret leaked in redacted: %s", prev.Redacted)
	}
	if prev.Unchanged {
		t.Fatal("expected changed when secret hit")
	}
	if len(prev.Spans) < 1 || len(prev.Segments) < 2 {
		t.Fatalf("expected spans/segments: spans=%d segs=%d", len(prev.Spans), len(prev.Segments))
	}
	rec, err := svc.IngestConfirm(ctx, core.IngestConfirmInput{
		Redacted: prev.Redacted, Filename: "note.md", Title: "Ingest secret sample", Format: prev.Format,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.UID == "" || rec.Trust != "proposed" {
		t.Fatalf("expected proposed experience: %+v", rec)
	}
}

func TestIngestPreviewDOCXBase64(t *testing.T) {
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

	docx := minimalDOCX(t, "林宇加班 手机号测试")
	prev, err := svc.IngestPreview(ctx, core.IngestPreviewInput{
		Filename:      "林宇-10月-加班表.docx",
		ContentBase64: base64.StdEncoding.EncodeToString(docx),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prev.Format != "docx" {
		t.Fatalf("format=%s", prev.Format)
	}
	if strings.Contains(prev.Original, "PK") {
		t.Fatalf("zip leaked into original preview")
	}
	if !strings.Contains(prev.Original, "加班") {
		t.Fatalf("missing extracted text: %q", prev.Original)
	}
	if !prev.Unchanged {
		t.Fatalf("expected unchanged for clean docx: hits=%v", prev.Hits)
	}
}

func TestIngestRejectsBinaryAsPlain(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_, _ = st.EnsureLocalUser(ctx)
	svc := core.NewService(st)
	_, err = svc.IngestPreview(ctx, core.IngestPreviewInput{
		Filename: "x.md",
		Text:     "PK\x03\x04" + strings.Repeat("\x00\x01\x02", 100),
	})
	if err == nil {
		t.Fatal("expected error")
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

func minimalDOCX(t *testing.T, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>` + body + `</w:t></w:r></w:p></w:body>
</w:document>`
	if _, err := w.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
