package extract_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/extract"
)

func TestDetectRejectsDOCXAsText(t *testing.T) {
	// ZIP magic without going through FromBytes plain path
	data := []byte("PK\x03\x04not-a-real-zip-but-binary")
	_, err := extract.Detect("notes.md", data)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func TestExtractDOCX(t *testing.T) {
	data := minimalDOCX(t, "加班表 姓名：测试 api_key=shouldstayinxml")
	res, err := extract.FromBytes("林宇-加班表.docx", data)
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != extract.FormatDOCX {
		t.Fatalf("format %s", res.Format)
	}
	if !strings.Contains(res.Text, "加班表") || !strings.Contains(res.Text, "姓名") {
		t.Fatalf("missing chinese text: %q", res.Text)
	}
	if strings.Contains(res.Text, "PK") {
		t.Fatalf("raw zip leaked: %q", res.Text)
	}
}

func TestExtractDOCXTableTSV(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>部门</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>技术部</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>姓名</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>林宇</w:t></w:r></w:p></w:tc>
      </w:tr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>日期</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>2026-10-30</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>时长</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>2</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	res, err := extract.FromBytes("sheet.docx", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "部门\t技术部\t姓名\t林宇") {
		t.Fatalf("expected TSV row: %q", res.Text)
	}
	if !strings.Contains(res.Text, "日期\t2026-10-30\t时长\t2") {
		t.Fatalf("expected second row: %q", res.Text)
	}
}

func TestExtractPlainSecretSafe(t *testing.T) {
	res, err := extract.FromBytes("a.txt", []byte("hello\nworld\n"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != extract.FormatPlain || res.CharCount < 5 {
		t.Fatalf("%+v", res)
	}
}

func TestChunk(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("段落内容足够长用于分块测试ABCDEFGHIJ\n\n")
	}
	chunks := extract.Chunk(b.String(), 80)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
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
