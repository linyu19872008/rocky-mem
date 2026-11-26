package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/store"
)

// ReindexResult summarises a search_doc rebuild.
type ReindexResult struct {
	MemoryIndexed     int `json:"memory_indexed"`
	ExperienceIndexed int `json:"experience_indexed"`
	ChunkIndexed      int `json:"chunk_indexed"`
	Deleted           int `json:"deleted_before"`
}

// Reindex rebuilds the derived search layer from business tables.
func (s *Service) Reindex(ctx context.Context) (*ReindexResult, error) {
	var out ReindexResult
	err := s.Store.WithTx(ctx, func(st store.Store) error {
		n, err := st.ClearSearchDocs(ctx)
		if err != nil {
			return err
		}
		out.Deleted = n
		mems, err := st.ListAllActiveMemory(ctx)
		if err != nil {
			return err
		}
		for _, m := range mems {
			if err := st.WriteSearchDoc(ctx, store.SearchDoc{
				Kind: "memory", OwnerID: m.ID, Lang: m.Lang, Title: m.Title, Body: memorySearchBody(m),
			}); err != nil {
				return err
			}
			out.MemoryIndexed++
		}
		exps, err := st.ListIndexableExperiences(ctx)
		if err != nil {
			return err
		}
		for _, e := range exps {
			if err := st.WriteSearchDoc(ctx, store.SearchDoc{
				Kind: "experience", OwnerID: e.ID, Lang: e.Lang, Title: e.Title, Body: experienceSearchBody(e),
			}); err != nil {
				return err
			}
			out.ExperienceIndexed++
		}
		chunks, err := st.ListAllActiveChunks(ctx)
		if err != nil {
			return err
		}
		titleBySource := map[int64]string{}
		if len(chunks) > 0 {
			seen := map[int64]bool{}
			var sids []int64
			for _, c := range chunks {
				if !seen[c.SourceID] {
					seen[c.SourceID] = true
					sids = append(sids, c.SourceID)
				}
			}
			if srcs, err := st.ListSourcesByIDs(ctx, sids); err == nil {
				for _, src := range srcs {
					titleBySource[src.ID] = src.Title
				}
			}
		}
		for _, c := range chunks {
			if err := st.WriteSearchDoc(ctx, store.SearchDoc{
				Kind: "chunk", OwnerID: c.ID, Lang: "und",
				Title: chunkDocTitle(titleBySource[c.SourceID], c.Heading), Body: c.Text,
			}); err != nil {
				return err
			}
			out.ChunkIndexed++
		}
		_ = st.InsertAudit(ctx, "write", "meta", nil, "",
			fmt.Sprintf("reindex:m=%d:e=%d:c=%d", out.MemoryIndexed, out.ExperienceIndexed, out.ChunkIndexed))
		return nil
	})
	return &out, err
}

// PurgeInput selects what to delete.
type PurgeInput struct {
	UID     string `json:"uid,omitempty"`
	Source  string `json:"source,omitempty"`  // s_… deletes the document and its chunks
	Before  int64  `json:"before,omitempty"`  // unix seconds; purge rows created before this
	Project string `json:"project,omitempty"` // project id decimal or empty
}

// PurgeResult counts deletions.
type PurgeResult struct {
	Memories     int `json:"memories"`
	Experiences  int `json:"experiences"`
	Sources      int `json:"sources"`
	Chunks       int `json:"chunks"`
	SearchDocs   int `json:"search_docs"`
	EvidenceRows int `json:"evidence"`
}

// Purge removes entries by uid, age, or project scope_ref.
func (s *Service) Purge(ctx context.Context, in PurgeInput) (*PurgeResult, error) {
	if in.UID == "" && in.Source == "" && in.Before == 0 && in.Project == "" {
		return nil, errs.New(errs.FieldRequired, "specify --uid, --source, --before, or --project", nil)
	}
	var out PurgeResult
	err := s.Store.WithTx(ctx, func(st store.Store) error {
		if in.Source != "" {
			src, err := st.GetSourceByUID(ctx, in.Source)
			if err != nil {
				return err
			}
			if src == nil {
				return errs.New(errs.NotFound, "source not found", map[string]any{"uid": in.Source})
			}
			n, err := st.DeleteSource(ctx, src.ID)
			if err != nil {
				return err
			}
			out.Sources, out.Chunks, out.SearchDocs = 1, n, n
			_ = st.InsertAudit(ctx, "purge", "source", &src.ID, src.UID, "purge_source")
			return nil
		}
		res, err := st.Purge(ctx, store.PurgeFilter{
			UID: in.UID, Before: in.Before, ProjectScopeRef: in.Project,
		})
		if err != nil {
			return err
		}
		out = PurgeResult{
			Memories: res.Memories, Experiences: res.Experiences,
			SearchDocs: res.SearchDocs, EvidenceRows: res.Evidence,
		}
		_ = st.InsertAudit(ctx, "purge", "store", nil, in.UID, fmt.Sprintf("before=%d project=%s", in.Before, in.Project))
		return nil
	})
	return &out, err
}

// ExportResult is L1 methodology export.
type ExportResult struct {
	Format string `json:"format"`
	Body   string `json:"body"`
	Count  int    `json:"count"`
}

// ExportL1 exports active/proposed experiences with sensitivity<=1 as methodology text.
// Level L1 whitelist: no code dumps — title, applies/invalid, fix summary only.
func (s *Service) ExportL1(ctx context.Context, format string) (*ExportResult, error) {
	if format == "" {
		format = "md"
	}
	exps, err := s.Store.ListExperiencesByTrust(ctx, []string{"active", "proposed"}, 5000)
	if err != nil {
		return nil, err
	}
	var kept []store.Experience
	for _, e := range exps {
		// L1 export: methodology fields only (no raw/evidence). Include sensitivity ≤ 2.
		if e.Sensitivity > 2 {
			continue
		}
		kept = append(kept, e)
	}
	var b strings.Builder
	switch format {
	case "jsonl":
		for _, e := range kept {
			b.WriteString(fmt.Sprintf(
				`{"uid":%q,"exp_type":%q,"title":%q,"applies_when":%q,"invalid_when":%q,"fix":%q}`+"\n",
				e.UID, e.ExpType, e.Title, e.AppliesWhen, e.InvalidWhen, trunc(e.FixMD, 500),
			))
		}
	default:
		b.WriteString("# rocky-mem export (L1 methodology)\n\n")
		b.WriteString(fmt.Sprintf("_exported %s_\n\n", time.Now().UTC().Format(time.RFC3339)))
		for _, e := range kept {
			b.WriteString(fmt.Sprintf("## [%s] %s\n", e.ExpType, e.Title))
			if e.AppliesWhen != "" {
				b.WriteString(fmt.Sprintf("- Applies when: %s\n", e.AppliesWhen))
			}
			if e.InvalidWhen != "" {
				b.WriteString(fmt.Sprintf("- Do not apply when: %s\n", e.InvalidWhen))
			}
			if e.FixMD != "" {
				b.WriteString(fmt.Sprintf("- Do: %s\n", trunc(e.FixMD, 500)))
			}
			b.WriteString(fmt.Sprintf("- uid: %s\n\n", e.UID))
		}
	}
	_ = s.Store.InsertAudit(ctx, "export", "store", nil, "", fmt.Sprintf("level=L1 format=%s count=%d", format, len(kept)))
	return &ExportResult{Format: format, Body: b.String(), Count: len(kept)}, nil
}
