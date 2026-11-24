<div align="center">

# rocky-mem

**A memory and experience service for coding agents.** Local-first by default,
self-hosted for teams.

Memory is state. Experience is a ledger. Both are searchable in one call.

[Design](design/DESIGN.md) · [Schema](design/schema.md) · [MCP tools](docs/mcp-tools.md) · [ADRs](design/adr/) · [中文](README.zh-CN.md)

</div>

---

> **Status: Tier-3 + Web SPA skeleton.** Six MCP tools, `serve` with cookie-auth
> React UI (en/zh-CN, light/dark), store conformance, backup/eval/term-map.
> Verify with `make verify-tier3`. See [docs/tier3.md](docs/tier3.md) and
> [docs/web.md](docs/web.md). Tagged `v0.1.0-tier3`. PostgreSQL / ingest remain later.

## Why

Every agent session starts from zero. Your last session's debugging path — the
commands that failed, the reason the fix worked, the condition under which it
stops working — is gone. What survives is a commit, which records *what* changed
and nothing about *why*.

Existing tools each capture a fragment:

- Chat transcripts record who said what, not what was done.
- Agent memory files keep conclusions and lose the process.
- `git log` covers code and nothing else.

rocky-mem keeps the triangle — **action, outcome, reason** — and makes it
retrievable by an agent in under 30 ms.

## Two modes, one binary

| | Solo | Team |
|---|---|---|
| Storage | SQLite (one file) | PostgreSQL |
| Network | loopback only | configurable bind |
| Identity | one implicit local user | users, teams, API tokens |
| Backup | `VACUUM INTO` snapshot | your database server |
| Start it with | `rockymem serve --dsn sqlite://…` | `rockymem serve --dsn postgres://…` |

Same binary, same commands, same MCP surface. The DSN is the only difference.

## What it looks like

An agent calls one tool before starting work:

```json
{ "query": "config lost after rolling update", "cwd": "/Users/x/repo/service-a" }
```

and gets a budgeted, sectioned pack with provenance:

```markdown
## Applies
### [pitfall] ConfigMap hot-reload is lost during rolling updates
- Applies when: K8s + file-mounted ConfigMap + rollout faster than 60s
- Do: add a checksum annotation to the pod template to force recreation
- Do not apply when: config is read through the ConfigMap API, not a mounted file
- Evidence: commit abc1234 (L2 · active · confidence 0.85 · e_01J9K…)

## Do not use
- `kubectl rollout restart` alone: still loses config (tried, failed · e_01JA…)
```

The "Do not use" section is the point. Agents rarely fail from not knowing a fix;
they fail from applying it outside its conditions.

## Principles

1. **Two kinds of knowledge, two tables.** Preference and fact are state (one
   current value). Experience is a ledger (accumulating claims that can be
   contested). One table cannot honour both write semantics.
2. **One write funnel, many entrances.** Agent writes, human notes and file drops
   pass the same redaction, deduplication and gating pipeline.
3. **Retrieval returns a context pack, not a result list.** Budgeted, sectioned,
   provenance-bearing.
4. **Evidence or it did not happen.** Experience requires a pointer to an
   immutable raw record.
5. **Append-only core, rebuildable indexes.** `reindex` reconstructs everything
   derived.
6. **Rules first, LLM last.** No LLM in the hot path.
7. **Explicit dialect divergence.** Two storage backends, one interface, separate
   reviewed SQL per dialect — not one magic query that pretends they are the same.
8. **Failures are first-class.** A `howto` that did not work is as valuable as one
   that did.
9. **Degradation is announced.** If FTS5 or `pg_trgm` is unavailable, `doctor` and
   the UI say so. Never silent.
10. **Removable.** Solo mode is one directory. `rm -rf` and nothing remains.

## Features (shipped vs planned)

**Shipped (SQLite solo):**

- **MCP server** — six tools. Prefer `make serve` first; MCP/CLI reuse the daemon
  via discovery files next to the DB (same `--dsn`).
- **Two knowledge types** — versioned memory with stable lineages; an experience
  ledger with trust states, conflict detection and falsification.
- **CLI ops** — `review`, `reindex`, `purge`, `export`, `backup`/`make backup`, `restore`, `eval`.
- **Retrieval** — FTS5 trigram + short-query LIKE + zh→en term-map expansion.
- **Web SPA** — React search / review / entry / settings; en·zh-CN; light/dark (`make web`).
- **Idempotent writes** — remember/record content dedupe; memory `rev` CAS.

**Planned:**

- **Richer Web UI** — conflict resolution, file ingest with redaction preview, i18n chrome.
- **PostgreSQL team mode** — same MCP surface; conformance suite on both backends.

## Architecture at a glance

```
MCP (Cursor, DSH) ─┐
HTTP /api/v1 ──────┼─▶ daemon ─▶ ingest funnel ─▶ store ─▶ SQLite / PostgreSQL
Web UI (SSE) ──────┘             retrieve pipeline ─▶ context pack
```

One daemon owns the database. MCP clients are thin proxies — Cursor and DSH each
spawn their own subprocess, and a subprocess must never open the database
directly. See [ADR-0005](design/adr/0005-technology-stack.md).

## Quick start (this repo)

```bash
make build
./scripts/seed-demo.sh          # demo prefs + global-scoped experiences
make serve                      # exclusive writer on .local/rocky-mem.db
# another terminal / Cursor MCP uses the same DSN and talks HTTP to serve
make verify-tier1
```

Project MCP config: [`.cursor/mcp.json`](.cursor/mcp.json) (same DSN as seed/serve).

```bash
# optional: brew install later
rockymem init --dsn sqlite://~/.local/share/rocky-mem/rocky-mem.db
rockymem serve --dsn sqlite://~/.local/share/rocky-mem/rocky-mem.db
```

## Non-goals

Screen recording and keystroke capture · hosted SaaS · multi-tenant isolation
between untrusted organisations · productivity reporting · translating your notes
· a general-purpose note app · graph visualizations · MySQL support.

The productivity one matters most: a store used for surveillance gets routed
around, and a biased sample is worse than no sample.

## Documentation

| | |
|---|---|
| [design/DESIGN.md](design/DESIGN.md) | Design: goals, principles, architecture, mechanisms, roadmap (Chinese) |
| [design/schema.md](design/schema.md) | Dual-dialect DDL, indexes, retrieval contract, migration (Chinese) |
| [docs/mcp-tools.md](docs/mcp-tools.md) | Agent-facing contract |
| [docs/error-codes.md](docs/error-codes.md) | Stable error registry |
| [docs/i18n.md](docs/i18n.md) | Localization and multilingual content |
| [docs/tier1.md](docs/tier1.md) | Daemon, review/export/eval gate |
| [docs/tier0-adoption.md](docs/tier0-adoption.md) | Agent adoption protocol |
| [design/adr/](design/adr/) | Why the irreversible choices were made (Chinese) |
| [design/README.md](design/README.md) | Design documentation index, reading order, maintenance rules |
| [docs/README.md](docs/README.md) | Contract documents index; where each kind of document belongs |

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first — in
particular the rule that a schema or query change must land in **both** dialect
directories in the same pull request.

Source code, comments, commit messages and contracts are **English**; design
documents are **Chinese**. Bug reports and feature requests are welcome in English
or Chinese.

## License

[Apache-2.0](LICENSE). Copyright is retained by contributors.
