package core

import (
	"context"
	"sort"
	"strings"
)

// BrowseInput lists memories and/or experiences for the library UI.
type BrowseInput struct {
	Kinds  []string `json:"kinds,omitempty"`  // memory | experience; default both
	Trusts []string `json:"trusts,omitempty"` // experience trusts; default ["active"]
	Q      string   `json:"q,omitempty"`      // case-insensitive title/preview filter
	Limit  int      `json:"limit,omitempty"`
}

// BrowseItem is one library card.
type BrowseItem struct {
	UID       string `json:"uid"`
	Kind      string `json:"kind"` // memory | experience
	Title     string `json:"title"`
	Trust     string `json:"trust,omitempty"`
	Subtype   string `json:"subtype,omitempty"` // mem_type or exp_type
	Preview   string `json:"preview,omitempty"`
	Pinned    bool   `json:"pinned,omitempty"`
	UpdatedAt int64  `json:"updated_at"`
}

// BrowseResult is the library listing payload.
type BrowseResult struct {
	Items []BrowseItem `json:"items"`
	Total int          `json:"total"`
}

// Browse returns recent memories/experiences without requiring a search query.
func (s *Service) Browse(ctx context.Context, in BrowseInput) (*BrowseResult, error) {
	kinds := normalizeBrowseKinds(in.Kinds)
	trusts := in.Trusts
	if len(trusts) == 0 {
		trusts = []string{"active"}
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 40
	}
	if limit > 200 {
		limit = 200
	}
	q := strings.ToLower(strings.TrimSpace(in.Q))

	var items []BrowseItem
	wantMem := kinds["memory"]
	wantExp := kinds["experience"]

	if wantMem {
		mems, err := s.Store.ListAllActiveMemory(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range mems {
			preview := trunc(strings.TrimSpace(m.BodyMD), 160)
			it := BrowseItem{
				UID: m.UID, Kind: "memory", Title: m.Title, Trust: m.Status,
				Subtype: m.MemType, Preview: preview, Pinned: m.Pinned, UpdatedAt: m.CreatedAt,
			}
			if m.TLastUsed != nil && *m.TLastUsed > it.UpdatedAt {
				it.UpdatedAt = *m.TLastUsed
			}
			if q != "" && !browseMatch(q, it.Title, it.Preview, it.Subtype) {
				continue
			}
			items = append(items, it)
		}
	}

	if wantExp {
		// Fetch extra then filter so title filter still fills the page.
		fetch := limit * 3
		if fetch < 60 {
			fetch = 60
		}
		if fetch > 500 {
			fetch = 500
		}
		exps, err := s.Store.ListExperiencesByTrust(ctx, trusts, fetch)
		if err != nil {
			return nil, err
		}
		for _, e := range exps {
			preview := firstNonEmpty(e.Symptom, e.FixMD, e.AppliesWhen)
			it := BrowseItem{
				UID: e.UID, Kind: "experience", Title: e.Title, Trust: e.Trust,
				Subtype: e.ExpType, Preview: trunc(preview, 160), UpdatedAt: e.TCreated,
			}
			if e.TLastUsed != nil && *e.TLastUsed > it.UpdatedAt {
				it.UpdatedAt = *e.TLastUsed
			}
			if q != "" && !browseMatch(q, it.Title, it.Preview, it.Subtype) {
				continue
			}
			items = append(items, it)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
	total := len(items)
	if len(items) > limit {
		items = items[:limit]
	}
	return &BrowseResult{Items: items, Total: total}, nil
}

func normalizeBrowseKinds(kinds []string) map[string]bool {
	out := map[string]bool{}
	for _, k := range kinds {
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "memory", "memories", "m":
			out["memory"] = true
		case "experience", "experiences", "e", "exp":
			out["experience"] = true
		}
	}
	if len(out) == 0 {
		out["memory"] = true
		out["experience"] = true
	}
	return out
}

func browseMatch(q, title, preview, subtype string) bool {
	return strings.Contains(strings.ToLower(title), q) ||
		strings.Contains(strings.ToLower(preview), q) ||
		strings.Contains(strings.ToLower(subtype), q)
}

func firstNonEmpty(parts ...string) string {
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			return p
		}
	}
	return ""
}
