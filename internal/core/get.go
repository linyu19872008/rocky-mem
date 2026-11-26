package core

import (
	"context"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/store"
)

// GetInput matches mem_get.
type GetInput struct {
	UID        string `json:"uid"`
	IncludeRaw bool   `json:"include_raw,omitempty"`
}

// GetResult is a full entry with evidence.
type GetResult struct {
	Type     string           `json:"type"` // memory | experience | source | chunk
	UID      string           `json:"uid"`
	Entry    map[string]any   `json:"entry"`
	Evidence []map[string]any `json:"evidence"`
	Tags     []string         `json:"tags"`
	Related  []map[string]any `json:"related,omitempty"`
	Lineage  any              `json:"lineage"`
	Versions []map[string]any `json:"versions,omitempty"`
	Chunks   []map[string]any `json:"chunks,omitempty"` // source only: chunk directory
	// Resolutions is the adjudication history of this entry's conflict group,
	// oldest first. Append-only: the last row is the current ruling.
	Resolutions []map[string]any `json:"resolutions,omitempty"`
}

// Get returns one memory or experience by uid (or memory lineage).
func (s *Service) Get(ctx context.Context, in GetInput) (*GetResult, error) {
	uid := strings.TrimSpace(in.UID)
	if uid == "" {
		return nil, errs.New(errs.FieldRequired, "uid is required", map[string]any{"field": "uid"})
	}

	if strings.HasPrefix(uid, "e_") {
		return s.getExperience(ctx, uid, in.IncludeRaw)
	}
	if strings.HasPrefix(uid, "s_") {
		return s.getSource(ctx, uid)
	}
	if strings.HasPrefix(uid, "c_") {
		return s.getChunk(ctx, uid)
	}
	// memory uid or lineage (lineage is typically an m_ uid)
	return s.getMemory(ctx, uid, in.IncludeRaw)
}

func (s *Service) getExperience(ctx context.Context, uid string, includeRaw bool) (*GetResult, error) {
	e, err := s.Store.GetExperienceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errs.New(errs.NotFound, "experience not found", map[string]any{"uid": uid})
	}
	ev, err := s.loadEvidence(ctx, "experience", e.ID, includeRaw)
	if err != nil {
		return nil, err
	}
	tags, _ := s.Store.ListTags(ctx, "experience", e.ID)
	var related []map[string]any
	if e.ConflictGroup != "" {
		peers, _ := s.Store.FindExperiencesByConflictGroup(ctx, e.ConflictGroup)
		for _, p := range peers {
			if p.UID == e.UID {
				continue
			}
			related = append(related, map[string]any{
				"uid": p.UID, "relation": "contradicts", "title": p.Title,
				"trust": p.Trust, "exp_type": p.ExpType, "confidence": p.Confidence,
				"symptom": p.Symptom, "fix_md": p.FixMD,
				"applies_when": p.AppliesWhen, "invalid_when": p.InvalidWhen,
				"outcome": p.Outcome,
			})
		}
	}
	var resolutions []map[string]any
	if e.ConflictGroup != "" {
		if rs, err := s.Store.ListResolutions(ctx, e.ConflictGroup); err == nil {
			for _, r := range rs {
				resolutions = append(resolutions, map[string]any{
					"action": r.Action, "note": r.Note, "created_at": r.CreatedAt,
					"kept_id": r.KeptID, "review_after": r.ReviewAfter,
				})
			}
		}
	}
	_ = s.Store.InsertAudit(ctx, "read", "experience", &e.ID, e.UID, "mem_get")
	return &GetResult{
		Type: "experience",
		UID:  e.UID,
		Entry: map[string]any{
			"exp_type": e.ExpType, "title": e.Title, "symptom": e.Symptom,
			"root_cause": e.RootCause, "fix_md": e.FixMD,
			"applies_when": e.AppliesWhen, "invalid_when": e.InvalidWhen,
			"outcome": e.Outcome, "confidence": e.Confidence, "trust": e.Trust,
			"conflict_group": nullIfEmpty(e.ConflictGroup),
			"scope":          e.Scope, "scope_ref": e.ScopeRef,
			"lang": e.Lang, "sensitivity": e.Sensitivity,
			"t_created": e.TCreated, "use_count": e.UseCount, "rev": e.Rev,
		},
		Evidence:    ev,
		Tags:        tags,
		Related:     related,
		Lineage:     nil,
		Resolutions: resolutions,
	}, nil
}

func (s *Service) getMemory(ctx context.Context, uid string, includeRaw bool) (*GetResult, error) {
	m, err := s.Store.GetMemoryByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	lineage := uid
	if m != nil {
		lineage = m.Lineage
		if m.Status != "active" {
			active, err := s.Store.GetActiveMemoryByLineage(ctx, m.Lineage)
			if err != nil {
				return nil, err
			}
			if active != nil {
				m = active
			}
		}
	} else {
		active, err := s.Store.GetActiveMemoryByLineage(ctx, uid)
		if err != nil {
			return nil, err
		}
		if active == nil {
			return nil, errs.New(errs.NotFound, "memory not found", map[string]any{"uid": uid})
		}
		m = active
		lineage = active.Lineage
	}

	ev, err := s.loadEvidence(ctx, "memory", m.ID, includeRaw)
	if err != nil {
		return nil, err
	}
	tags, _ := s.Store.ListTags(ctx, "memory", m.ID)
	versions, _ := s.Store.ListMemoryVersions(ctx, lineage)
	var verOut []map[string]any
	for _, v := range versions {
		verOut = append(verOut, map[string]any{
			"uid": v.UID, "status": v.Status, "created_at": v.CreatedAt, "rev": v.Rev,
		})
	}
	var memKey any
	if m.MemKey != nil {
		memKey = *m.MemKey
	}
	_ = s.Store.InsertAudit(ctx, "read", "memory", &m.ID, m.UID, "mem_get")
	return &GetResult{
		Type: "memory",
		UID:  m.UID,
		Entry: map[string]any{
			"mem_key": memKey, "mem_type": m.MemType, "title": m.Title, "body_md": m.BodyMD,
			"status": m.Status, "pinned": m.Pinned, "priority": m.Priority,
			"lineage": m.Lineage, "scope": m.Scope, "scope_ref": m.ScopeRef,
			"lang": m.Lang, "sensitivity": m.Sensitivity,
			"created_at": m.CreatedAt, "use_count": m.UseCount, "rev": m.Rev,
		},
		Evidence: ev,
		Tags:     tags,
		Lineage:  m.Lineage,
		Versions: verOut,
	}, nil
}

func (s *Service) loadEvidence(ctx context.Context, ownerType string, ownerID int64, includeRaw bool) ([]map[string]any, error) {
	rows, err := s.Store.ListEvidence(ctx, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, ev := range rows {
		item := map[string]any{
			"role": ev.Role, "quote": ev.Quote,
		}
		raw, err := s.Store.GetRaw(ctx, ev.RawID)
		if err != nil {
			return nil, err
		}
		if raw != nil {
			item["raw_uid"] = raw.UID
			item["channel"] = raw.Channel
			item["sensitivity"] = raw.Sensitivity
			if includeRaw {
				body := string(raw.Body)
				item["excerpt"] = trunc(body, 500)
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// getSource returns a document's metadata plus a directory of its chunks.
// Chunk text is not included: that is what being background material means.
func (s *Service) getSource(ctx context.Context, uid string) (*GetResult, error) {
	src, err := s.Store.GetSourceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, errs.New(errs.NotFound, "source not found", map[string]any{"uid": uid})
	}
	chunks, err := s.Store.ListChunksBySource(ctx, src.ID)
	if err != nil {
		return nil, err
	}
	dir := make([]map[string]any, 0, len(chunks))
	for _, c := range chunks {
		dir = append(dir, map[string]any{
			"uid": c.UID, "ord": c.Ord, "heading": nullIfEmpty(c.Heading),
			"char_count": c.CharCount, "preview": trunc(oneLine(c.Text), 120),
		})
	}
	_ = s.Store.InsertAudit(ctx, "read", "source", &src.ID, src.UID, "mem_get")
	return &GetResult{
		Type: "source",
		UID:  src.UID,
		Entry: map[string]any{
			"kind": src.Kind, "uri": src.URI, "title": src.Title, "format": src.Format,
			"char_count": src.CharCount, "chunk_count": src.ChunkCount, "lang": src.Lang,
			"sensitivity": src.Sensitivity, "status": src.Status,
			"superseded_by": src.SupersededBy, "project_id": src.ProjectID,
			"created_at": src.CreatedAt, "use_count": src.UseCount,
		},
		Chunks: dir,
	}, nil
}

// getChunk returns one slice with its source metadata, so an agent that only
// has a c_… uid from a pack can read the passage in full.
func (s *Service) getChunk(ctx context.Context, uid string) (*GetResult, error) {
	c, err := s.Store.GetChunkByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errs.New(errs.NotFound, "chunk not found", map[string]any{"uid": uid})
	}
	src, _ := s.sourceByID(ctx, c.SourceID)
	entry := map[string]any{
		"ord": c.Ord, "heading": nullIfEmpty(c.Heading), "char_count": c.CharCount,
		"sensitivity": c.Sensitivity, "text": c.Text,
	}
	if src != nil {
		entry["source_uid"] = src.UID
		entry["source_title"] = src.Title
		entry["source_status"] = src.Status
	}
	_ = s.Store.InsertAudit(ctx, "read", "chunk", &c.ID, c.UID, "mem_get")
	return &GetResult{Type: "chunk", UID: c.UID, Entry: entry}, nil
}

func (s *Service) sourceByID(ctx context.Context, id int64) (*store.Source, error) {
	got, err := s.Store.ListSourcesByIDs(ctx, []int64{id})
	if err != nil || len(got) == 0 {
		return nil, err
	}
	return &got[0], nil
}
