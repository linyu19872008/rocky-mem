package core

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/redact"
	"github.com/rocky-mem/rocky-mem/internal/textnorm"
)

// IngestPreviewInput is phase-1 of two-phase file/text ingest.
type IngestPreviewInput struct {
	Text     string `json:"text"`
	Filename string `json:"filename,omitempty"`
}

// IngestPreviewResult shows redaction before persist.
type IngestPreviewResult struct {
	Filename    string         `json:"filename,omitempty"`
	ContentHash string         `json:"content_hash"`
	Original    string         `json:"original"`
	Redacted    string         `json:"redacted"`
	Hits        map[string]int `json:"hits,omitempty"`
	Level       int            `json:"sensitivity_level"`
	Rejected    bool           `json:"rejected"`
	RejectWhy   string         `json:"reject_why,omitempty"`
	RulePack    string         `json:"rule_pack"`
}

// IngestConfirmInput is phase-2: persist the already-previewed redacted text.
type IngestConfirmInput struct {
	Redacted string `json:"redacted"`
	Filename string `json:"filename,omitempty"`
	Title    string `json:"title,omitempty"`
	Cwd      string `json:"cwd,omitempty"`
}

// IngestPreview runs the redact rule pack without writing.
func (s *Service) IngestPreview(_ context.Context, in IngestPreviewInput) (*IngestPreviewResult, error) {
	text := strings.ReplaceAll(in.Text, "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return nil, errs.New(errs.FieldRequired, "text required", nil)
	}
	if len([]rune(text)) > 200_000 {
		return nil, errs.New(errs.FieldInvalid, "text too large (max ~200k runes)", nil)
	}
	rr := redact.Apply(text)
	return &IngestPreviewResult{
		Filename:    strings.TrimSpace(in.Filename),
		ContentHash: textnorm.Hash(text),
		Original:    text,
		Redacted:    rr.Text,
		Hits:        rr.Hits,
		Level:       rr.Level,
		Rejected:    rr.Rejected,
		RejectWhy:   rr.RejectWhy,
		RulePack:    rr.RulePack,
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
		Evidence:    []EvidenceIn{{Type: "quote", Text: evText, Ref: strings.TrimSpace(in.Filename)}},
	})
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
