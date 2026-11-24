// Package core implements remember / record / search / context pack assembly.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/project"
	"github.com/rocky-mem/rocky-mem/internal/redact"
	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/textnorm"
	"github.com/rocky-mem/rocky-mem/internal/uid"
)

// Service is the application API used by CLI and MCP.
type Service struct {
	Store store.Store
}

// ScopeResolved is returned to callers after cwd → project mapping.
type ScopeResolved struct {
	Scope     string `json:"scope"`
	ProjectID int64  `json:"project_id,omitempty"`
	Project   string `json:"project,omitempty"`
	ScopeRef  string `json:"scope_ref,omitempty"`
}

func (s *Service) resolveProject(ctx context.Context, cwd string) (*store.Project, *ScopeResolved, error) {
	if cwd == "" {
		return nil, &ScopeResolved{Scope: "global", ScopeRef: ""}, nil
	}
	res, err := project.Resolve(cwd)
	if err != nil || res == nil {
		return nil, nil, errs.New(errs.ProjectUnresolved, "cannot resolve project from cwd", map[string]any{"cwd": cwd})
	}
	uidLocal, err := s.Store.EnsureLocalUser(ctx)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.Store.UpsertProject(ctx, store.Project{
		Name:          res.Name,
		RemoteURLNorm: res.RemoteURLNorm,
		RootHint:      res.RootHint,
		CreatedBy:     &uidLocal,
	})
	if err != nil {
		return nil, nil, err
	}
	return &p, &ScopeResolved{
		Scope:     "project",
		ProjectID: p.ID,
		Project:   p.Name,
		ScopeRef:  strconv.FormatInt(p.ID, 10),
	}, nil
}

func readScopes(resolved *ScopeResolved) []store.ScopeFilter {
	out := []store.ScopeFilter{{Scope: "global", ScopeRef: ""}}
	if resolved != nil && resolved.Scope == "project" {
		out = append(out, store.ScopeFilter{Scope: "project", ScopeRef: resolved.ScopeRef})
	}
	return out
}

func writeScope(explicitScope, explicitRef string, resolved *ScopeResolved) (scope, ref string) {
	if explicitScope != "" {
		scope = explicitScope
		ref = explicitRef
		if scope == "global" {
			ref = ""
		}
		if scope == "project" && ref == "" && resolved != nil {
			ref = resolved.ScopeRef
		}
		return scope, ref
	}
	if resolved != nil && resolved.Scope == "project" {
		return "project", resolved.ScopeRef
	}
	// Never infer global on write failure path — session fallback.
	return "session", "default"
}

func memorySearchBody(m store.Memory) string {
	return m.Title + "\n" + m.BodyMD
}

func experienceSearchBody(e store.Experience) string {
	parts := []string{e.Title, e.Symptom, e.RootCause, e.FixMD, e.AppliesWhen, e.InvalidWhen}
	return strings.Join(parts, "\n")
}

func redactJSON(hits map[string]int) string {
	if len(hits) == 0 {
		return ""
	}
	b, _ := json.Marshal(hits)
	return string(b)
}

func (s *Service) putRaw(ctx context.Context, text, channel, sourceRef, cwd string, projectID *int64) (store.Raw, error) {
	rr := redact.Apply(text)
	if rr.Rejected {
		return store.Raw{}, errs.New(errs.PolicyRejected, rr.RejectWhy, nil)
	}
	body := textnorm.NFC(rr.Text)
	h := textnorm.Hash(body)
	if existing, err := s.Store.GetRawByHash(ctx, h); err != nil {
		return store.Raw{}, err
	} else if existing != nil {
		return *existing, nil
	}
	actor, _ := s.Store.EnsureLocalUser(ctx)
	return s.Store.InsertRaw(ctx, store.Raw{
		UID:         uid.Raw(),
		Hash:        h,
		Channel:     channel,
		SourceRef:   sourceRef,
		ActorID:     &actor,
		Agent:       "rockymem",
		Cwd:         cwd,
		ProjectID:   projectID,
		Lang:        "und",
		Body:        []byte(body),
		Redaction:   redactJSON(rr.Hits),
		RulePack:    rr.RulePack,
		Sensitivity: rr.Level,
	})
}

func qualityGate(e store.Experience, evidenceCount int) (trust, status, reason string, review bool) {
	var reasons []string
	if evidenceCount < 1 {
		reasons = append(reasons, "evidence missing")
	}
	if strings.TrimSpace(e.AppliesWhen) == "" {
		reasons = append(reasons, "applies_when missing")
	}
	if e.Confidence < 0.6 {
		reasons = append(reasons, "confidence below 0.6")
	}
	if e.Sensitivity > 2 {
		reasons = append(reasons, "sensitivity > 2")
	}
	if len(reasons) > 0 {
		return "proposed", "pending_review", strings.Join(reasons, "; "), true
	}
	return "active", "published", "", false
}

func approxTokens(s string) int {
	// Rough: ~4 chars per token for mixed CJK/Latin.
	n := len([]rune(s))
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func fmtMemLine(m store.Memory) string {
	pin := ""
	if m.Pinned {
		pin = " · pinned"
	}
	return fmt.Sprintf("- %s（%s%s · %s）", strings.TrimSpace(m.BodyMD), m.Scope, pin, m.UID)
}

func fmtExpBlock(e store.Experience, evidenceNote string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("### [%s] %s\n", e.ExpType, e.Title))
	if e.AppliesWhen != "" {
		b.WriteString(fmt.Sprintf("- Applies when: %s\n", e.AppliesWhen))
	}
	if e.FixMD != "" {
		b.WriteString(fmt.Sprintf("- Do: %s\n", trunc(e.FixMD, 400)))
	}
	if e.InvalidWhen != "" {
		b.WriteString(fmt.Sprintf("- Do not apply when: %s\n", e.InvalidWhen))
	}
	if evidenceNote != "" {
		b.WriteString(fmt.Sprintf("- Evidence: %s (%s · %s · confidence %.2f · %s)\n",
			evidenceNote, sensLabel(e.Sensitivity), e.Trust, e.Confidence, e.UID))
	} else {
		b.WriteString(fmt.Sprintf("- Evidence: (none listed) (%s · %s · confidence %.2f · %s)\n",
			sensLabel(e.Sensitivity), e.Trust, e.Confidence, e.UID))
	}
	return b.String()
}

func sensLabel(n int) string {
	switch n {
	case 1:
		return "L1"
	case 3:
		return "L3"
	default:
		return "L2"
	}
}

func trustWeight(t string) float64 {
	switch t {
	case "active":
		return 1.0
	case "contested":
		return 0.7
	case "proposed":
		return 0.5
	case "stale":
		return 0.4
	case "falsified":
		return 0.1
	default:
		return 0.3
	}
}
