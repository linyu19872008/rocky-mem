package adoption

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/core"
)

// WorkflowResult is the outcome of a Tier-0 adoption simulation.
type WorkflowResult struct {
	StartedAt       int64    `json:"started_at"`
	ContextUIDs     []string `json:"context_uids"`
	SearchUIDs      []string `json:"search_uids"`
	RememberUID     string   `json:"remember_uid"`
	RecordUID       string   `json:"record_uid"`
	RecordTrust     string   `json:"record_trust"`
	FeedbackEffect  string   `json:"feedback_effect"`
	FoundAfterWrite bool     `json:"found_after_write"`
	GetType         string   `json:"get_type"`
	Steps           []string `json:"steps"`
}

// RunWorkflow exercises the adoption loop against a core.Service:
// context → search → remember → record → search → get → feedback.
// marker is a unique token embedded in written entries so the run is isolatable.
func RunWorkflow(ctx context.Context, svc *core.Service, cwd, marker string) (*WorkflowResult, error) {
	if marker == "" {
		marker = fmt.Sprintf("tier0-%d", time.Now().UnixNano())
	}
	out := &WorkflowResult{StartedAt: time.Now().Unix()}
	step := func(s string) { out.Steps = append(out.Steps, s) }

	ctxRes, err := svc.Context(ctx, core.ContextInput{Cwd: cwd, BudgetTokens: 600})
	if err != nil {
		return out, fmt.Errorf("mem_context: %w", err)
	}
	out.ContextUIDs = ctxRes.UIDs
	step("mem_context")

	search1, err := svc.Search(ctx, core.SearchInput{
		Query: marker, Cwd: cwd, IncludeProposed: true, BudgetTokens: 800,
	})
	if err != nil {
		return out, fmt.Errorf("mem_search(initial): %w", err)
	}
	out.SearchUIDs = search1.UIDs
	step("mem_search:initial")

	rem, err := svc.Remember(ctx, core.RememberInput{
		Title:   "tier0 adoption preference",
		Body:    "Tier-0 workflow marker preference: " + marker,
		MemKey:  "tier0_adoption_" + sanitizeKey(marker),
		MemType: "pref", Scope: "session", Cwd: cwd, Pinned: true,
	})
	if err != nil {
		return out, fmt.Errorf("mem_remember: %w", err)
	}
	out.RememberUID = rem.UID
	step("mem_remember")

	rec, err := svc.Record(ctx, core.RecordInput{
		Title:       "tier0 adoption lesson " + marker,
		ExpType:     "howto",
		Symptom:     "need to verify agent adoption loop " + marker,
		RootCause:   "tier-0 harness requires a round-trip write",
		Fix:         "run adoption workflow then inspect audit",
		AppliesWhen: "after changing MCP protocol or adoption rules",
		InvalidWhen: "when only editing unrelated docs",
		Outcome:     "success",
		Confidence:  0.85,
		Scope:       "session",
		Cwd:         cwd,
		Evidence: []core.EvidenceIn{{
			Type: "quote",
			Text: "tier0 adoption evidence " + marker,
		}},
	})
	if err != nil {
		return out, fmt.Errorf("mem_record: %w", err)
	}
	out.RecordUID = rec.UID
	out.RecordTrust = rec.Trust
	step("mem_record")

	search2, err := svc.Search(ctx, core.SearchInput{
		Query: marker, Cwd: cwd, IncludeProposed: true, BudgetTokens: 1200,
	})
	if err != nil {
		return out, fmt.Errorf("mem_search(after write): %w", err)
	}
	for _, u := range search2.UIDs {
		if u == out.RecordUID || u == out.RememberUID {
			out.FoundAfterWrite = true
			break
		}
	}
	if !out.FoundAfterWrite && strings.Contains(search2.Pack, marker) {
		out.FoundAfterWrite = true
	}
	step("mem_search:after_write")

	got, err := svc.Get(ctx, core.GetInput{UID: out.RecordUID, IncludeRaw: true})
	if err != nil {
		return out, fmt.Errorf("mem_get: %w", err)
	}
	out.GetType = got.Type
	step("mem_get")

	fb, err := svc.Feedback(ctx, core.FeedbackInput{
		UID: out.RecordUID, Outcome: "helpful",
	})
	if err != nil {
		return out, fmt.Errorf("mem_feedback: %w", err)
	}
	out.FeedbackEffect = fb.Effect
	step("mem_feedback")

	if !out.FoundAfterWrite {
		return out, fmt.Errorf("written entries not found by mem_search for marker %q", marker)
	}
	if out.GetType != "experience" {
		return out, fmt.Errorf("mem_get type=%q want experience", out.GetType)
	}
	return out, nil
}

func sanitizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// RequiredWorkflowSteps is the ordered adoption loop.
var RequiredWorkflowSteps = []string{
	"mem_context",
	"mem_search:initial",
	"mem_remember",
	"mem_record",
	"mem_search:after_write",
	"mem_get",
	"mem_feedback",
}
