package core

import (
	"context"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
)

// GetInput matches mem_get.
type GetInput struct {
	UID        string `json:"uid"`
	IncludeRaw bool   `json:"include_raw,omitempty"`
}

// GetResult is a full entry with evidence.
type GetResult struct {
	Type     string           `json:"type"` // memory | experience
	UID      string           `json:"uid"`
	Entry    map[string]any   `json:"entry"`
	Evidence []map[string]any `json:"evidence"`
	Tags     []string         `json:"tags"`
	Related  []map[string]any `json:"related,omitempty"`
	Lineage  any              `json:"lineage"`
	Versions []map[string]any `json:"versions,omitempty"`
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
			})
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
			"scope": e.Scope, "scope_ref": e.ScopeRef,
			"lang": e.Lang, "sensitivity": e.Sensitivity,
			"t_created": e.TCreated, "use_count": e.UseCount, "rev": e.Rev,
		},
		Evidence: ev,
		Tags:     tags,
		Related:  related,
		Lineage:  nil,
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
