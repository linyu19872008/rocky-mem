# Tier-0 adoption — delivery contract

Tier-0 proves the fatal design assumption: agents can discover and use rocky-mem.

## Deliverables

| ID | Deliverable | Verification |
|---|---|---|
| 0.1 | `AGENTS.md` + `.cursor/rules/rocky-mem-mcp.mdc` | `go test ./internal/adoption -run Protocol` |
| 0.2 | Automated adoption workflow (context→search→write→get→feedback) | `go test ./internal/adoption -run Workflow` |
| 0.3 | Audit-based verdict + checklist | `go test ./internal/adoption -run Audit` + `make verify-tier0` |

## One-command gate

```bash
make verify-tier0
```

This runs:

1. `go test ./internal/adoption ./internal/core …`
2. `rockymem adopt-verify --write-verdict`
3. Expect exit 0 and `status: **pass**` in `docs/adoption-verdict.md`

## Manual live-agent checklist (required for human sign-off)

Automated tests cannot prove Cursor will *voluntarily* call tools. After `make verify-tier0`:

1. Open a **new** Agent chat in this repo.
2. Ask **one** non-trivial question (排查 / 设计 / 修 bug 类均可；例如 “查一下 ConfigMap 滚动更新相关经验并总结”).  
   Do **not** use trivial prompts (“把变量改个名”). The same question is enough to cover the whole checklist — no need for separate tasks per box.
3. Confirm the transcript shows `mem_context` and/or `mem_search` **without** you naming the tool.
4. Tick the boxes in `docs/adoption-verdict.md`.

## Pass / fail rules

| Condition | Result |
|---|---|
| Protocol files missing markers | **fail** |
| Workflow round-trip fails | **fail** |
| Protocol + workflow OK | **pass** (automated); live checklist still recommended |
| Live agent never calls MCP | **product risk** — update rule wording, re-run checklist |

## Related

- Design risk: `design/DESIGN.md` §16–§17
- MCP contracts: `docs/mcp-tools.md`
