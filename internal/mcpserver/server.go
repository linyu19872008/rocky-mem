// Package mcpserver registers rocky-mem MCP tools.
package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/version"
)

// New builds an MCP server bound to a MemAPI (local Service or HTTP client).
func New(api core.MemAPI) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "rocky-mem",
		Version: version.Version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "mem_context",
		Description: "Return standing preferences for a working directory under a small fixed budget. " +
			"Cheap enough to call unconditionally on every session.",
	}, makeContext(api))

	mcp.AddTool(server, &mcp.Tool{
		Name: "mem_search",
		Description: "Search the local knowledge base before starting a non-trivial task. " +
			"Returns a budgeted context pack with provenance. If an entry conflicts with what you " +
			"actually observe, call mem_feedback with outcome=wrong instead of silently ignoring it.",
	}, makeSearch(api))

	mcp.AddTool(server, &mcp.Tool{
		Name: "mem_get",
		Description: "Fetch one memory or experience by uid (m_… / e_…), including evidence chain. " +
			"For memory, always resolves to the current active version in the lineage.",
	}, makeGet(api))

	mcp.AddTool(server, &mcp.Tool{
		Name: "mem_record",
		Description: "Record a reusable engineering lesson. Evidence is mandatory: cite what you " +
			"actually observed. Entries are reviewed before they become authoritative; the response " +
			"tells you the real status — do not claim success if status is pending_review or rejected.",
	}, makeRecord(api))

	mcp.AddTool(server, &mcp.Tool{
		Name: "mem_remember",
		Description: "Write or update a standing memory (preference, fact, constraint). " +
			"Providing mem_key enables upsert semantics for the current value.",
	}, makeRemember(api))

	mcp.AddTool(server, &mcp.Tool{
		Name: "mem_feedback",
		Description: "Report whether a retrieved entry was helpful, wrong, or stale. " +
			"Call freely; outcome=wrong demotes experience trust immediately.",
	}, makeFeedback(api))

	return server
}

func toolJSON(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}

func toolErr(err error) (*mcp.CallToolResult, any, error) {
	e := errs.As(err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: e.JSON()}},
		IsError: true,
	}, nil, nil
}

type contextArgs struct {
	Cwd          string `json:"cwd,omitempty" jsonschema:"Absolute path of the working directory"`
	TaskHint     string `json:"task_hint,omitempty" jsonschema:"Optional one-line task description"`
	BudgetTokens int    `json:"budget_tokens,omitempty" jsonschema:"Token budget, default 600"`
}

func makeContext(api core.MemAPI) func(context.Context, *mcp.CallToolRequest, contextArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in contextArgs) (*mcp.CallToolResult, any, error) {
		out, err := api.Context(ctx, core.ContextInput{
			Cwd: in.Cwd, TaskHint: in.TaskHint, BudgetTokens: in.BudgetTokens,
		})
		if err != nil {
			return toolErr(err)
		}
		return toolJSON(out), out, nil
	}
}

type searchArgs struct {
	Query           string   `json:"query" jsonschema:"Search query"`
	Cwd             string   `json:"cwd,omitempty"`
	Scope           string   `json:"scope,omitempty"`
	Kinds           []string `json:"kinds,omitempty"`
	Lang            string   `json:"lang,omitempty"`
	IncludeProposed *bool    `json:"include_proposed,omitempty"`
	BudgetTokens    int      `json:"budget_tokens,omitempty"`
	Limit           int      `json:"limit,omitempty"`
}

func makeSearch(api core.MemAPI) func(context.Context, *mcp.CallToolRequest, searchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in searchArgs) (*mcp.CallToolResult, any, error) {
		include := true
		if in.IncludeProposed != nil {
			include = *in.IncludeProposed
		}
		out, err := api.Search(ctx, core.SearchInput{
			Query: in.Query, Cwd: in.Cwd, Scope: in.Scope, Kinds: in.Kinds,
			Lang: in.Lang, IncludeProposed: include, BudgetTokens: in.BudgetTokens, Limit: in.Limit,
		})
		if err != nil {
			return toolErr(err)
		}
		return toolJSON(out), out, nil
	}
}

type evidenceArg struct {
	Type string `json:"type"`
	Ref  string `json:"ref,omitempty"`
	Text string `json:"text,omitempty"`
}

type recordArgs struct {
	Title       string        `json:"title"`
	ExpType     string        `json:"exp_type"`
	Symptom     string        `json:"symptom,omitempty"`
	RootCause   string        `json:"root_cause,omitempty"`
	Fix         string        `json:"fix,omitempty"`
	AppliesWhen string        `json:"applies_when,omitempty"`
	InvalidWhen string        `json:"invalid_when,omitempty"`
	Outcome     string        `json:"outcome,omitempty"`
	Confidence  float64       `json:"confidence,omitempty"`
	Scope       string        `json:"scope,omitempty"`
	ScopeRef    string        `json:"scope_ref,omitempty"`
	Evidence    []evidenceArg `json:"evidence"`
	Tags        []string      `json:"tags,omitempty"`
	Cwd         string        `json:"cwd,omitempty"`
}

func makeRecord(api core.MemAPI) func(context.Context, *mcp.CallToolRequest, recordArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in recordArgs) (*mcp.CallToolResult, any, error) {
		ev := make([]core.EvidenceIn, len(in.Evidence))
		for i, e := range in.Evidence {
			ev[i] = core.EvidenceIn{Type: e.Type, Ref: e.Ref, Text: e.Text}
		}
		out, err := api.Record(ctx, core.RecordInput{
			Title: in.Title, ExpType: in.ExpType, Symptom: in.Symptom, RootCause: in.RootCause,
			Fix: in.Fix, AppliesWhen: in.AppliesWhen, InvalidWhen: in.InvalidWhen,
			Outcome: in.Outcome, Confidence: in.Confidence, Scope: in.Scope, ScopeRef: in.ScopeRef,
			Evidence: ev, Tags: in.Tags, Cwd: in.Cwd,
		})
		if err != nil {
			return toolErr(err)
		}
		return toolJSON(out), out, nil
	}
}

type rememberArgs struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	MemKey      string   `json:"mem_key,omitempty"`
	MemType     string   `json:"mem_type,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	ScopeRef    string   `json:"scope_ref,omitempty"`
	Pinned      bool     `json:"pinned,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Rev         *int     `json:"rev,omitempty" jsonschema:"Optimistic-lock token from mem_get; omit for last-write-wins"`
	Tags        []string `json:"tags,omitempty"`
	SourceQuote string   `json:"source_quote,omitempty"`
	Cwd         string   `json:"cwd,omitempty"`
}

func makeRemember(api core.MemAPI) func(context.Context, *mcp.CallToolRequest, rememberArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in rememberArgs) (*mcp.CallToolResult, any, error) {
		out, err := api.Remember(ctx, core.RememberInput{
			Title: in.Title, Body: in.Body, MemKey: in.MemKey, MemType: in.MemType,
			Scope: in.Scope, ScopeRef: in.ScopeRef, Pinned: in.Pinned, Priority: in.Priority,
			Rev: in.Rev, Tags: in.Tags, SourceQuote: in.SourceQuote, Cwd: in.Cwd,
		})
		if err != nil {
			return toolErr(err)
		}
		return toolJSON(out), out, nil
	}
}

type getArgs struct {
	UID        string `json:"uid" jsonschema:"m_… / e_… / memory lineage uid"`
	IncludeRaw bool   `json:"include_raw,omitempty" jsonschema:"Include truncated evidence excerpts"`
}

func makeGet(api core.MemAPI) func(context.Context, *mcp.CallToolRequest, getArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in getArgs) (*mcp.CallToolResult, any, error) {
		out, err := api.Get(ctx, core.GetInput{UID: in.UID, IncludeRaw: in.IncludeRaw})
		if err != nil {
			return toolErr(err)
		}
		return toolJSON(out), out, nil
	}
}

type feedbackArgs struct {
	UID     string `json:"uid"`
	Outcome string `json:"outcome" jsonschema:"helpful|wrong|stale"`
	Note    string `json:"note,omitempty" jsonschema:"Required when outcome=wrong"`
}

func makeFeedback(api core.MemAPI) func(context.Context, *mcp.CallToolRequest, feedbackArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in feedbackArgs) (*mcp.CallToolResult, any, error) {
		out, err := api.Feedback(ctx, core.FeedbackInput{
			UID: in.UID, Outcome: in.Outcome, Note: in.Note,
		})
		if err != nil {
			return toolErr(err)
		}
		return toolJSON(out), out, nil
	}
}
