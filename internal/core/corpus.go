package core

import (
	"context"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/redact"
	"github.com/linyu19872008/rocky-mem/internal/store"
	"github.com/linyu19872008/rocky-mem/internal/textnorm"
	"github.com/linyu19872008/rocky-mem/internal/uid"
)

// Ingest modes. Corpus is the default: a document is background material, not
// a claim. Distill additionally proposes one experience for review.
const (
	ModeCorpus  = "corpus"
	ModeDistill = "distill"

	// chunkMaxRunes bounds one chunk. Long paragraphs are hard-split so no
	// chunk exceeds it; retrieval quality depends on chunks staying comparable.
	chunkMaxRunes = 3500
)

// IngestConfirmResult reports what phase-2 stored. When mode=distill, the
// record fields are populated; in corpus mode they are empty and UID is empty.
type IngestConfirmResult struct {
	Mode             string `json:"mode"`
	SourceUID        string `json:"source_uid"`
	SourceTitle      string `json:"source_title,omitempty"`
	ChunkCount       int    `json:"chunk_count"`
	ArchivedPrevious int    `json:"archived_previous,omitempty"`
	Unchanged        bool   `json:"unchanged,omitempty"`

	// Distill only.
	UID            string `json:"uid,omitempty"`
	Rev            int    `json:"rev,omitempty"`
	Status         string `json:"status,omitempty"`
	Trust          string `json:"trust,omitempty"`
	ReviewRequired bool   `json:"review_required,omitempty"`
}

type chunkPiece struct {
	Heading string
	Text    string
}

// IngestConfirm persists the previewed, already-redacted text as a source plus
// its chunks, then optionally proposes one experience for review.
//
// Corpus is not evidence: no raw row is created. Distillation runs after the
// corpus transaction commits, so a distill failure never loses the document.
func (s *Service) IngestConfirm(ctx context.Context, in IngestConfirmInput) (*IngestConfirmResult, error) {
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
	body = strings.TrimSpace(rr.Text)

	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode == "" {
		mode = ModeCorpus
	}
	if mode != ModeCorpus && mode != ModeDistill {
		return nil, errs.New(errs.FieldInvalid, "mode must be corpus or distill", map[string]any{"mode": mode})
	}

	title := ingestTitle(in)
	uri := ingestURI(in)
	format := strings.TrimSpace(in.Format)
	if format == "" {
		format = "plain"
	}
	hash := textnorm.Hash(body)
	level := rr.Level
	if level <= 0 {
		level = 2
	}

	actor, err := s.Store.EnsureLocalUser(ctx)
	if err != nil {
		return nil, err
	}
	var projectID *int64
	if p, _, perr := s.resolveProject(ctx, in.Cwd); perr == nil && p != nil {
		projectID = &p.ID
	}

	pieces := splitChunks(body, chunkMaxRunes)
	res := &IngestConfirmResult{Mode: mode, ChunkCount: len(pieces), SourceTitle: title}

	// Same content already ingested: reuse the source rather than duplicating
	// it. Distillation still runs, because "re-drop the doc, now also distill"
	// is a legitimate follow-up; Record's own dedupe keeps it idempotent.
	if existing, err := s.Store.GetSourceByTextHash(ctx, hash); err == nil && existing != nil {
		res.SourceUID = existing.UID
		res.SourceTitle = existing.Title
		res.ChunkCount = existing.ChunkCount
		res.Unchanged = true
	} else if err := s.writeCorpus(ctx, in, body, hash, title, uri, format, level, actor, projectID, pieces, res); err != nil {
		return nil, err
	}

	if mode == ModeDistill {
		rec, rerr := s.Record(ctx, RecordInput{
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
			Evidence:    []EvidenceIn{{Type: "quote", Text: trunc(body, 2000), Ref: uri + " (" + format + ")"}},
		})
		if rerr != nil {
			return nil, rerr
		}
		res.UID = rec.UID
		res.Rev = rec.Rev
		res.Status = rec.Status
		res.Trust = rec.Trust
		res.ReviewRequired = rec.ReviewRequired
	}
	return res, nil
}

func (s *Service) writeCorpus(ctx context.Context, in IngestConfirmInput, body, hash, title, uri, format string,
	level int, actor int64, projectID *int64, pieces []chunkPiece, res *IngestConfirmResult) error {
	return s.Store.WithTx(ctx, func(tx store.Store) error {
		src, err := tx.InsertSource(ctx, store.Source{
			UID: uid.Source(), Kind: ingestKind(in), URI: uri, Title: title,
			Format: format, TextHash: hash, CharCount: utf8.RuneCountInString(body),
			ChunkCount: len(pieces), Lang: "und", Sensitivity: level,
			Status: "active", ProjectID: projectID, ActorID: &actor,
		})
		if err != nil {
			return err
		}

		chunks := make([]store.Chunk, len(pieces))
		for i, p := range pieces {
			chunks[i] = store.Chunk{
				UID: uid.Chunk(), SourceID: src.ID, Ord: i, Heading: p.Heading,
				Text: p.Text, CharCount: utf8.RuneCountInString(p.Text), Sensitivity: level,
			}
		}
		chunks, err = tx.InsertChunks(ctx, chunks)
		if err != nil {
			return err
		}
		for _, c := range chunks {
			if err := tx.WriteSearchDoc(ctx, store.SearchDoc{
				Kind: "chunk", OwnerID: c.ID, Lang: "und",
				Title: chunkDocTitle(src.Title, c.Heading), Body: c.Text,
			}); err != nil {
				return err
			}
		}

		// A re-ingest of the same uri with different content archives the old
		// version, so retrieval never returns two generations at once.
		archived, err := tx.ArchiveSourceByURI(ctx, projectID, uri, src.ID)
		if err != nil {
			return err
		}
		res.ArchivedPrevious = archived
		res.SourceUID = src.UID
		res.SourceTitle = src.Title
		res.ChunkCount = len(chunks)
		return nil
	})
}

func ingestKind(in IngestConfirmInput) string {
	if strings.TrimSpace(in.Filename) == "" {
		return "paste"
	}
	return "file"
}

func ingestURI(in IngestConfirmInput) string {
	base := filepath.Base(strings.TrimSpace(in.Filename))
	if base == "" || base == "." || base == "/" {
		return "paste"
	}
	return base
}

func ingestTitle(in IngestConfirmInput) string {
	if t := strings.TrimSpace(in.Title); t != "" {
		return t
	}
	base := ingestURI(in)
	if base != "paste" {
		if ext := filepath.Ext(base); ext != "" {
			if stem := strings.TrimSpace(strings.TrimSuffix(base, ext)); stem != "" {
				return stem
			}
		}
		return base
	}
	return "Ingest note"
}

func chunkDocTitle(sourceTitle, heading string) string {
	if heading == "" {
		return sourceTitle
	}
	return sourceTitle + " · " + heading
}

// splitChunks packs paragraphs up to maxRunes and starts a new chunk at every
// markdown heading. Headings are tracked per chunk; non-markdown documents get
// an empty heading.
func splitChunks(text string, maxRunes int) []chunkPiece {
	if maxRunes <= 0 {
		maxRunes = chunkMaxRunes
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	paras := strings.Split(text, "\n\n")

	var out []chunkPiece
	var cur strings.Builder
	curRunes := 0
	heading := ""

	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			out = append(out, chunkPiece{Heading: heading, Text: s})
		}
		cur.Reset()
		curRunes = 0
	}

	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if h, ok := markdownHeading(p); ok {
			if curRunes > 0 {
				flush()
			}
			heading = h
		}
		n := utf8.RuneCountInString(p)
		if n > maxRunes {
			if curRunes > 0 {
				flush()
			}
			for _, piece := range hardSplit(p, maxRunes) {
				out = append(out, chunkPiece{Heading: heading, Text: piece})
			}
			continue
		}
		if curRunes > 0 && curRunes+n > maxRunes {
			flush()
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(p)
		curRunes += n
	}
	flush()
	return out
}

// markdownHeading reports whether the paragraph starts with an ATX heading.
func markdownHeading(p string) (string, bool) {
	line := p
	if i := strings.IndexByte(p, '\n'); i >= 0 {
		line = p[:i]
	}
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	rest := strings.TrimLeft(line, "#")
	if !strings.HasPrefix(rest, " ") {
		return "", false // "#hashtag" is not a heading
	}
	h := strings.TrimSpace(rest)
	if h == "" {
		return "", false
	}
	return h, true
}

func hardSplit(p string, maxRunes int) []string {
	runes := []rune(p)
	var out []string
	for len(runes) > maxRunes {
		out = append(out, strings.TrimSpace(string(runes[:maxRunes])))
		runes = runes[maxRunes:]
	}
	if rest := strings.TrimSpace(string(runes)); rest != "" {
		out = append(out, rest)
	}
	return out
}
