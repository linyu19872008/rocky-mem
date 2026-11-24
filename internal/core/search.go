package core

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/uid"
)

// ContextInput matches mem_context.
type ContextInput struct {
	Cwd          string `json:"cwd,omitempty"`
	TaskHint     string `json:"task_hint,omitempty"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// SearchInput matches mem_search.
type SearchInput struct {
	Query           string   `json:"query"`
	Cwd             string   `json:"cwd,omitempty"`
	Scope           string   `json:"scope,omitempty"`
	Kinds           []string `json:"kinds,omitempty"`
	Lang            string   `json:"lang,omitempty"`
	IncludeProposed bool     `json:"include_proposed"`
	BudgetTokens    int      `json:"budget_tokens,omitempty"`
	Limit           int      `json:"limit,omitempty"`
}

// PackResult is a budgeted context pack.
type PackResult struct {
	Pack          string         `json:"pack"`
	Sections      []SectionCount `json:"sections,omitempty"`
	UIDs          []string       `json:"uids"`
	Truncated     bool           `json:"truncated"`
	Omitted       int            `json:"omitted,omitempty"`
	ScopeResolved *ScopeResolved `json:"scope_resolved,omitempty"`
	Notice        *string        `json:"notice,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
}

// SectionCount summarises pack sections.
type SectionCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Context returns standing preferences for a working directory.
func (s *Service) Context(ctx context.Context, in ContextInput) (*PackResult, error) {
	if in.BudgetTokens <= 0 {
		in.BudgetTokens = 600
	}
	_, resolved, err := s.resolveProject(ctx, in.Cwd)
	if err != nil && in.Cwd != "" {
		n := "degraded:project_unresolved"
		resolved = &ScopeResolved{Scope: "global"}
		mems, _ := s.Store.ListActiveMemory(ctx, readScopes(resolved), 20)
		pack, uids, trunc, omitted := assemblePreferences(mems, nil, nil, in.BudgetTokens)
		return &PackResult{
			Pack: pack, UIDs: uids, Truncated: trunc, Omitted: omitted,
			ScopeResolved: resolved, Notice: &n, RequestID: uid.Request(),
			Sections: []SectionCount{{Name: "preferences", Count: len(uids)}},
		}, nil
	}
	mems, err := s.Store.ListActiveMemory(ctx, readScopes(resolved), 20)
	if err != nil {
		return nil, err
	}
	// Optional: light search on task hint for pinned-ish experiences.
	var exps []store.Experience
	if strings.TrimSpace(in.TaskHint) != "" {
		cands, _ := s.Store.SearchCandidates(ctx, in.TaskHint, "experience", 10)
		ids := ownerIDs(cands, "experience")
		exps, _ = s.Store.ListExperiencesByIDs(ctx, ids)
		exps = filterVisible(exps, true)
	}
	pack, uids, trunc, omitted := assemblePreferences(mems, exps, nil, in.BudgetTokens)
	_ = s.Store.InsertAudit(ctx, "read", "pack", nil, "", "mem_context")
	return &PackResult{
		Pack: pack, UIDs: uids, Truncated: trunc, Omitted: omitted,
		ScopeResolved: resolved, RequestID: uid.Request(),
		Sections: sectionCounts(pack),
	}, nil
}

// Search returns a sectioned context pack.
func (s *Service) Search(ctx context.Context, in SearchInput) (*PackResult, error) {
	if strings.TrimSpace(in.Query) == "" {
		return nil, fmt.Errorf("query required")
	}
	if in.BudgetTokens <= 0 {
		in.BudgetTokens = 1200
	}
	if in.Limit <= 0 {
		in.Limit = 12
	}
	in.IncludeProposed = true // default per contract

	_, resolved, _ := s.resolveProject(ctx, in.Cwd)
	if resolved == nil {
		resolved = &ScopeResolved{Scope: "global"}
	}

	kinds := in.Kinds
	if len(kinds) == 0 {
		kinds = []string{"memory", "experience"}
	}

	var mems []store.Memory
	var exps []store.Experience

	for _, k := range kinds {
		cands, err := s.Store.SearchCandidates(ctx, in.Query, k, 50)
		if err != nil {
			continue
		}
		switch k {
		case "memory":
			ids := ownerIDs(cands, "memory")
			got, _ := s.Store.ListMemoriesByIDs(ctx, ids)
			mems = append(mems, filterMemScope(got, readScopes(resolved))...)
		case "experience":
			ids := ownerIDs(cands, "experience")
			got, _ := s.Store.ListExperiencesByIDs(ctx, ids)
			exps = append(exps, rerankExperiences(got, in.Query, resolved, in.IncludeProposed)...)
		}
	}

	// Also pull standing prefs even if FTS missed them.
	standing, _ := s.Store.ListActiveMemory(ctx, readScopes(resolved), 10)
	mems = mergeMemories(standing, mems)

	if len(exps) > in.Limit {
		exps = exps[:in.Limit]
	}

	pack, uids, truncated, omitted := assemblePack(mems, exps, in.BudgetTokens)
	for _, m := range mems {
		_ = s.Store.BumpUse(ctx, "memory", m.ID, time.Now().Unix())
	}
	for _, e := range exps {
		_ = s.Store.BumpUse(ctx, "experience", e.ID, time.Now().Unix())
	}
	_ = s.Store.InsertAudit(ctx, "read", "pack", nil, "", "mem_search:"+in.Query)

	var notice *string
	if len(uids) == 0 {
		n := "empty_store_or_no_match: use mem_record to add experience"
		notice = &n
	}
	return &PackResult{
		Pack: pack, UIDs: uids, Truncated: truncated, Omitted: omitted,
		ScopeResolved: resolved, Notice: notice, RequestID: uid.Request(),
		Sections: sectionCounts(pack),
	}, nil
}

func ownerIDs(cands []store.Candidate, kind string) []int64 {
	var ids []int64
	seen := map[int64]struct{}{}
	for _, c := range cands {
		if c.Kind != kind {
			continue
		}
		if _, ok := seen[c.OwnerID]; ok {
			continue
		}
		seen[c.OwnerID] = struct{}{}
		ids = append(ids, c.OwnerID)
	}
	return ids
}

func filterVisible(exps []store.Experience, includeProposed bool) []store.Experience {
	var out []store.Experience
	for _, e := range exps {
		if e.Trust == "archived" || e.Trust == "falsified" {
			continue
		}
		if e.Trust == "proposed" && !includeProposed {
			continue
		}
		out = append(out, e)
	}
	return out
}

func filterMemScope(mems []store.Memory, scopes []store.ScopeFilter) []store.Memory {
	allow := map[string]struct{}{}
	for _, s := range scopes {
		allow[s.Scope+"|"+s.ScopeRef] = struct{}{}
	}
	var out []store.Memory
	for _, m := range mems {
		if m.Status != "active" {
			continue
		}
		if _, ok := allow[m.Scope+"|"+m.ScopeRef]; ok || m.Scope == "global" {
			out = append(out, m)
		}
	}
	return out
}

func mergeMemories(a, b []store.Memory) []store.Memory {
	seen := map[int64]struct{}{}
	var out []store.Memory
	for _, m := range append(a, b...) {
		if _, ok := seen[m.ID]; ok {
			continue
		}
		seen[m.ID] = struct{}{}
		out = append(out, m)
	}
	return out
}

func filterExpScope(exps []store.Experience, resolved *ScopeResolved) []store.Experience {
	var out []store.Experience
	for _, e := range exps {
		switch e.Scope {
		case "global", "session":
			out = append(out, e)
		case "project":
			if resolved != nil && e.ScopeRef == resolved.ScopeRef {
				out = append(out, e)
			}
		default:
			out = append(out, e)
		}
	}
	return out
}

func rerankExperiences(exps []store.Experience, query string, resolved *ScopeResolved, includeProposed bool) []store.Experience {
	exps = filterVisible(exps, includeProposed)
	exps = filterExpScope(exps, resolved)
	type scored struct {
		e store.Experience
		s float64
	}
	q := strings.ToLower(query)
	var list []scored
	for _, e := range exps {
		score := trustWeight(e.Trust)
		if resolved != nil && e.Scope == "project" && e.ScopeRef == resolved.ScopeRef {
			score += 0.55
		} else if e.Scope == "global" {
			score += 0.10
		}
		if strings.Contains(strings.ToLower(e.Title), q) {
			score += 0.40
		}
		if e.UseCount > 0 {
			score += math.Min(0.20, math.Log1p(float64(e.UseCount))*0.05)
		}
		list = append(list, scored{e, score})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].s > list[j].s })
	out := make([]store.Experience, len(list))
	for i, s := range list {
		out[i] = s.e
	}
	return out
}

func assemblePreferences(mems []store.Memory, exps []store.Experience, _ any, budget int) (string, []string, bool, int) {
	return assemblePack(mems, exps, budget)
}

func assemblePack(mems []store.Memory, exps []store.Experience, budget int) (string, []string, bool, int) {
	var prefs, applies, donot, bg []string
	var uids []string
	used := 0
	omitted := 0
	truncated := false

	add := func(section *[]string, block, id string) bool {
		t := approxTokens(block)
		if used+t > budget && used > 0 {
			omitted++
			truncated = true
			return false
		}
		*section = append(*section, block)
		uids = append(uids, id)
		used += t
		return true
	}

	for _, m := range mems {
		if !add(&prefs, fmtMemLine(m), m.UID) {
			break
		}
	}
	for _, e := range exps {
		if e.Outcome == "failed" {
			line := fmt.Sprintf("- %s: %s（试过，不行 · %s）", e.Title, trunc(e.Symptom, 120), e.UID)
			if !add(&donot, line, e.UID) {
				break
			}
			continue
		}
		block := fmtExpBlock(e, "see evidence chain")
		if !add(&applies, block, e.UID) {
			break
		}
	}

	var b strings.Builder
	if len(prefs) > 0 {
		b.WriteString("## Preferences\n")
		b.WriteString(strings.Join(prefs, "\n"))
		b.WriteByte('\n')
	}
	if len(applies) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("## Applies\n")
		b.WriteString(strings.Join(applies, "\n"))
		b.WriteByte('\n')
	}
	if len(donot) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("## Do not use\n")
		b.WriteString(strings.Join(donot, "\n"))
		b.WriteByte('\n')
	}
	if len(bg) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("## Background\n")
		b.WriteString(strings.Join(bg, "\n"))
		b.WriteByte('\n')
	}
	if truncated {
		b.WriteString(fmt.Sprintf("\n_Omitted %d items; call mem_get for individual entries._\n", omitted))
	}
	if b.Len() == 0 {
		b.WriteString("_No matching memory or experience._\n")
	}
	return b.String(), uids, truncated, omitted
}

func sectionCounts(pack string) []SectionCount {
	names := []string{"preferences", "applies", "do_not_use", "background"}
	heads := []string{"## Preferences", "## Applies", "## Do not use", "## Background"}
	var out []SectionCount
	for i, h := range heads {
		count := 0
		if idx := strings.Index(pack, h); idx >= 0 {
			rest := pack[idx+len(h):]
			end := len(rest)
			for _, other := range heads {
				if j := strings.Index(rest, other); j >= 0 && j < end {
					end = j
				}
			}
			section := rest[:end]
			count = strings.Count(section, "\n- ") + strings.Count(section, "\n### ")
		}
		out = append(out, SectionCount{Name: names[i], Count: count})
	}
	return out
}
