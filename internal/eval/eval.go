// Package eval runs a fixed retrieval corpus and reports recall@10.
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

// Corpus is the on-disk eval fixture.
type Corpus struct {
	Entries []Entry `json:"entries"`
	Queries []Query `json:"queries"`
}

// Entry is one experience to index.
type Entry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ExpType     string `json:"exp_type"`
	Symptom     string `json:"symptom"`
	RootCause   string `json:"root_cause"`
	Fix         string `json:"fix"`
	AppliesWhen string `json:"applies_when"`
	InvalidWhen string `json:"invalid_when"`
	Evidence    string `json:"evidence"`
}

// Query maps a search string to relevant entry IDs.
type Query struct {
	Q        string   `json:"q"`
	Relevant []string `json:"relevant"`
}

// Report summarises one eval run.
type Report struct {
	Queries    int     `json:"queries"`
	Hits       int     `json:"hits"`
	Misses     int     `json:"misses"`
	RecallAt10 float64 `json:"recall_at_10"`
	Details    []QDetail `json:"details,omitempty"`
}

// QDetail is per-query outcome.
type QDetail struct {
	Query   string   `json:"query"`
	Want    []string `json:"want"`
	GotIDs  []string `json:"got_ids"`
	Hit     bool     `json:"hit"`
}

// DefaultCorpusPath resolves testdata/eval/corpus.json next to this package.
func DefaultCorpusPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "testdata/eval/corpus.json"
	}
	return filepath.Join(filepath.Dir(file), "testdata", "corpus.json")
}

// RunFile loads corpus JSON and evaluates against a fresh temp store.
func RunFile(ctx context.Context, path string) (*Report, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Corpus
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return Run(ctx, c)
}

// Run indexes the corpus and measures recall@10 (hit if any relevant id appears in top pack).
func Run(ctx context.Context, c Corpus) (*Report, error) {
	dir, err := os.MkdirTemp("", "rocky-eval-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	st, err := sqlite.Open(filepath.Join(dir, "eval.db"))
	if err != nil {
		return nil, err
	}
	defer st.Close()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		return nil, err
	}
	svc := &core.Service{Store: st}

	idToUID := map[string]string{}
	for _, e := range c.Entries {
		expType := e.ExpType
		if expType == "" {
			expType = "pitfall"
		}
		applies := e.AppliesWhen
		if applies == "" {
			applies = "general engineering"
		}
		ev := e.Evidence
		if ev == "" {
			ev = "eval fixture: " + e.ID
		}
		rec, err := svc.Record(ctx, core.RecordInput{
			Title: e.Title, ExpType: expType, Symptom: e.Symptom, RootCause: e.RootCause,
			Fix: e.Fix, AppliesWhen: applies, InvalidWhen: e.InvalidWhen,
			Outcome: "success", Confidence: 0.9, Scope: "global",
			Evidence: []core.EvidenceIn{{Type: "quote", Text: ev}},
		})
		if err != nil {
			return nil, fmt.Errorf("index %s: %w", e.ID, err)
		}
		idToUID[e.ID] = rec.UID
		// Tag via search body already includes title; map uid→id through title match later.
	}

	uidToID := map[string]string{}
	for id, uid := range idToUID {
		uidToID[uid] = id
	}

	rep := &Report{Queries: len(c.Queries)}
	for _, q := range c.Queries {
		pack, err := svc.Search(ctx, core.SearchInput{
			Query: q.Q, IncludeProposed: true, BudgetTokens: 2000, Limit: 10,
		})
		if err != nil {
			return nil, err
		}
		got := make([]string, 0, len(pack.UIDs))
		gotSet := map[string]bool{}
		for _, uid := range pack.UIDs {
			if id, ok := uidToID[uid]; ok {
				got = append(got, id)
				gotSet[id] = true
			}
		}
		// Also match by title substring if UID mapping missed (budget truncation).
		for _, e := range c.Entries {
			if strings.Contains(pack.Pack, e.Title) {
				gotSet[e.ID] = true
			}
		}
		hit := false
		for _, want := range q.Relevant {
			if gotSet[want] {
				hit = true
				break
			}
		}
		d := QDetail{Query: q.Q, Want: q.Relevant, GotIDs: got, Hit: hit}
		rep.Details = append(rep.Details, d)
		if hit {
			rep.Hits++
		} else {
			rep.Misses++
		}
	}
	if rep.Queries > 0 {
		rep.RecallAt10 = float64(rep.Hits) / float64(rep.Queries)
	}
	return rep, nil
}
