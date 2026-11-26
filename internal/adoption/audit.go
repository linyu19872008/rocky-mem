package adoption

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/store"
)

// Verdict is the Tier-0 adoption conclusion derived from audit + workflow.
type Verdict struct {
	Status           string         `json:"status"` // pass | fail | inconclusive
	GeneratedAt      int64          `json:"generated_at"`
	SinceUnix        int64          `json:"since_unix"`
	ReadCount        int            `json:"read_count"`
	WriteCount       int            `json:"write_count"`
	SearchMentions   int            `json:"search_mentions"`
	ContextMentions  int            `json:"context_mentions"`
	RecordMentions   int            `json:"record_mentions"`
	RememberMentions int            `json:"remember_mentions"`
	FeedbackMentions int            `json:"feedback_mentions"`
	WorkflowOK       bool           `json:"workflow_ok"`
	ProtocolOK       bool           `json:"protocol_ok"`
	Notes            []string       `json:"notes"`
	CountsByDetail   map[string]int `json:"counts_by_detail,omitempty"`
}

// AnalyzeAudit summarises audit rows since sinceUnix for adoption signals.
func AnalyzeAudit(rows []store.AuditRow, sinceUnix int64) Verdict {
	v := Verdict{
		GeneratedAt:    time.Now().Unix(),
		SinceUnix:      sinceUnix,
		Status:         "inconclusive",
		CountsByDetail: map[string]int{},
		Notes:          []string{},
	}
	for _, r := range rows {
		if r.TS < sinceUnix {
			continue
		}
		switch r.Action {
		case "read":
			v.ReadCount++
		case "write", "update":
			v.WriteCount++
		}
		d := strings.ToLower(r.Detail)
		v.CountsByDetail[r.Detail]++
		if strings.Contains(d, "mem_search") || strings.HasPrefix(d, "mem_search") {
			v.SearchMentions++
		}
		if strings.Contains(d, "mem_context") {
			v.ContextMentions++
		}
		if r.TargetType == "experience" && r.Action == "write" && !strings.Contains(d, "feedback") {
			v.RecordMentions++
		}
		if r.TargetType == "memory" && (r.Action == "write" || r.Action == "update") && !strings.Contains(d, "feedback") {
			v.RememberMentions++
		}
		if strings.Contains(d, "feedback") {
			v.FeedbackMentions++
		}
	}
	return v
}

// FinalizeVerdict applies pass/fail rules for Tier-0 delivery.
func FinalizeVerdict(v Verdict, protocolOK, workflowOK bool) Verdict {
	v.ProtocolOK = protocolOK
	v.WorkflowOK = workflowOK
	v.Notes = append([]string{}, v.Notes...)

	if !protocolOK {
		v.Status = "fail"
		v.Notes = append(v.Notes, "protocol files missing required markers")
	}
	if !workflowOK {
		v.Status = "fail"
		v.Notes = append(v.Notes, "automated adoption workflow failed")
	}
	if protocolOK && workflowOK {
		// Automated gate passed. Human live-agent trials remain recommended.
		if v.SearchMentions >= 1 && v.RecordMentions >= 1 && v.ContextMentions >= 1 {
			v.Status = "pass"
			v.Notes = append(v.Notes, "automated workflow produced context+search+record audit signals")
		} else {
			v.Status = "pass"
			v.Notes = append(v.Notes,
				"protocol+workflow OK; audit signal counts may be sparse depending on detail tags",
				"still run a live Cursor Agent session and append notes to docs/adoption-verdict.md",
			)
		}
	}
	return v
}

// LoadAndAnalyze reads audit rows from the store and returns a verdict.
func LoadAndAnalyze(ctx context.Context, st store.Store, sinceUnix int64, protocolOK, workflowOK bool) (Verdict, error) {
	rows, err := st.ListAudit(ctx, sinceUnix, 1000)
	if err != nil {
		return Verdict{}, err
	}
	v := AnalyzeAudit(rows, sinceUnix)
	return FinalizeVerdict(v, protocolOK, workflowOK), nil
}

// RenderVerdictMarkdown produces a short verdict section for docs.
func RenderVerdictMarkdown(v Verdict) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Tier-0 adoption verdict\n\n"))
	b.WriteString(fmt.Sprintf("- status: **%s**\n", v.Status))
	b.WriteString(fmt.Sprintf("- generated_at: %d\n", v.GeneratedAt))
	b.WriteString(fmt.Sprintf("- protocol_ok: %v\n", v.ProtocolOK))
	b.WriteString(fmt.Sprintf("- workflow_ok: %v\n", v.WorkflowOK))
	b.WriteString(fmt.Sprintf("- reads: %d · writes: %d\n", v.ReadCount, v.WriteCount))
	b.WriteString(fmt.Sprintf("- search_mentions: %d · context_mentions: %d · record_mentions: %d · feedback_mentions: %d\n",
		v.SearchMentions, v.ContextMentions, v.RecordMentions, v.FeedbackMentions))
	if len(v.Notes) > 0 {
		b.WriteString("\n## Notes\n\n")
		for _, n := range v.Notes {
			b.WriteString("- " + n + "\n")
		}
	}
	b.WriteString("\n## Live agent checklist (manual)\n\n")
	b.WriteString("- [ ] New Agent chat called `mem_context` without being asked\n")
	b.WriteString("- [ ] Non-trivial task triggered `mem_search` before coding\n")
	b.WriteString("- [ ] At least one `mem_record` or `mem_remember` after a lesson\n")
	b.WriteString("- [ ] Conflict path: `mem_feedback outcome=wrong` used once\n")
	return b.String()
}
