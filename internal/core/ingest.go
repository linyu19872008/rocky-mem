package core

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/extract"
	"github.com/rocky-mem/rocky-mem/internal/redact"
)

// IngestPreviewInput is phase-1 of two-phase ingest.
// Prefer ContentBase64 for files; Text is for paste-only plain content.
type IngestPreviewInput struct {
	Text          string `json:"text,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
	Filename      string `json:"filename,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
}

// IngestPreviewResult shows extraction + redaction before persist.
type IngestPreviewResult struct {
	Filename    string           `json:"filename,omitempty"`
	Format      string           `json:"format"`
	ContentHash string           `json:"content_hash"`
	CharCount   int              `json:"char_count"`
	ChunkCount  int              `json:"chunk_count"`
	Original    string           `json:"original"`
	Redacted    string           `json:"redacted"`
	Segments    []redact.Segment `json:"segments,omitempty"`
	Spans       []redact.Span    `json:"spans,omitempty"`
	Hits        map[string]int   `json:"hits,omitempty"`
	Level       int              `json:"sensitivity_level"`
	Rejected    bool             `json:"rejected"`
	RejectWhy   string           `json:"reject_why,omitempty"`
	RulePack    string           `json:"rule_pack"`
	Notice      string           `json:"notice,omitempty"`
	Unchanged   bool             `json:"unchanged"`
}

// IngestConfirmInput is phase-2: persist already-previewed redacted text.
type IngestConfirmInput struct {
	Redacted string `json:"redacted"`
	Filename string `json:"filename,omitempty"`
	Title    string `json:"title,omitempty"`
	Format   string `json:"format,omitempty"`
	Cwd      string `json:"cwd,omitempty"`
}

// IngestPreview extracts text (plain/docx/xlsx/pdf) then runs the redact pack.
func (s *Service) IngestPreview(_ context.Context, in IngestPreviewInput) (*IngestPreviewResult, error) {
	data, filename, err := resolveIngestBytes(in)
	if err != nil {
		return nil, err
	}
	ex, err := extract.FromBytes(filename, data)
	if err != nil {
		return nil, mapExtractErr(err)
	}
	rr := redact.Apply(ex.Text)
	chunks := extract.Chunk(ex.Text, 3500)
	notice := ""
	if ex.Format != extract.FormatPlain {
		notice = "extracted plain text from " + string(ex.Format) + "; original binary is never shown"
	}
	if rr.Unchanged {
		if notice != "" {
			notice += ". "
		}
		notice += "no redaction hits — preview equals extracted text"
	}
	return &IngestPreviewResult{
		Filename:    filename,
		Format:      string(ex.Format),
		ContentHash: ex.ContentHash,
		CharCount:   ex.CharCount,
		ChunkCount:  len(chunks),
		Original:    ex.Text,
		Redacted:    rr.Text,
		Segments:    rr.Segments,
		Spans:       rr.Spans,
		Hits:        rr.Hits,
		Level:       rr.Level,
		Rejected:    rr.Rejected,
		RejectWhy:   rr.RejectWhy,
		RulePack:    rr.RulePack,
		Notice:      notice,
		Unchanged:   rr.Unchanged,
	}, nil
}

// IngestConfirm writes a proposed experience from redacted body (+ evidence quote).
func (s *Service) IngestConfirm(ctx context.Context, in IngestConfirmInput) (*RecordResult, error) {
	body := strings.TrimSpace(in.Redacted)
	if body == "" {
		return nil, errs.New(errs.FieldRequired, "redacted text required", nil)
	}
	rr := redact.Apply(body)
	if rr.Rejected {
		return nil, errs.New(errs.FieldInvalid, "content rejected by redaction: "+rr.RejectWhy, map[string]any{
			"reject_why": rr.RejectWhy,
		})
	}
	body = rr.Text
	title := strings.TrimSpace(in.Title)
	if title == "" {
		base := filepath.Base(strings.TrimSpace(in.Filename))
		if base != "" && base != "." && base != "/" {
			title = "Ingest: " + base
		} else {
			title = "Ingest note"
		}
	}
	ref := strings.TrimSpace(in.Filename)
	if in.Format != "" {
		if ref != "" {
			ref = ref + " (" + in.Format + ")"
		} else {
			ref = in.Format
		}
	}
	evText := trunc(body, 2000)
	return s.Record(ctx, RecordInput{
		Title:       title,
		ExpType:     "howto",
		Symptom:     trunc(firstNonEmptyLine(body), 240),
		Fix:         trunc(body, 4000),
		AppliesWhen: "after human ingest review",
		InvalidWhen: "secrets still visible after redaction",
		Outcome:     "unknown",
		Confidence:  0.55,
		Scope:       "global",
		Cwd:         in.Cwd,
		Evidence:    []EvidenceIn{{Type: "quote", Text: evText, Ref: ref}},
	})
}

func resolveIngestBytes(in IngestPreviewInput) ([]byte, string, error) {
	filename := strings.TrimSpace(in.Filename)
	if in.ContentBase64 != "" {
		raw, err := base64.StdEncoding.DecodeString(in.ContentBase64)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(in.ContentBase64)
			if err != nil {
				return nil, "", errs.New(errs.FieldInvalid, "invalid content_base64", nil)
			}
		}
		if filename == "" {
			filename = "upload.bin"
		}
		return raw, filename, nil
	}
	text := strings.ReplaceAll(in.Text, "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return nil, "", errs.New(errs.FieldRequired, "text or content_base64 required", nil)
	}
	if filename == "" {
		filename = "paste.txt"
	}
	return []byte(text), filename, nil
}

func mapExtractErr(err error) error {
	if err == nil {
		return nil
	}
	if ee, ok := err.(*extract.Error); ok {
		return errs.New(errs.FieldInvalid, ee.Message, map[string]any{"extract": ee.Code})
	}
	return errs.New(errs.FieldInvalid, err.Error(), nil)
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return s
}
