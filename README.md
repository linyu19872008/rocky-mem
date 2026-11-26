<div align="center">

<img src="assets/brand/icon-512.png" width="132" alt="Rocky, the rocky-mem mascot">

# rocky-mem

**A memory and experience service for coding agents.** Local-first, solo by design.

Memory is state. Experience is a ledger. Both are searchable in one call.

[![ci](https://github.com/linyu19872008/rocky-mem/actions/workflows/ci.yml/badge.svg)](https://github.com/linyu19872008/rocky-mem/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/linyu19872008/rocky-mem?sort=semver)](https://github.com/linyu19872008/rocky-mem/releases)
[![license](https://img.shields.io/github/license/linyu19872008/rocky-mem)](LICENSE)
[![go](https://img.shields.io/github/go-mod/go-version/linyu19872008/rocky-mem)](go.mod)
[![skill](https://img.shields.io/npm/v/rocky-mem-skill?label=skill)](https://www.npmjs.com/package/rocky-mem-skill)


[Design](design/DESIGN.md) · [Schema](design/schema.md) · [MCP tools](docs/mcp-tools.md) · [ADRs](design/adr/) · [中文](README.zh-CN.md)

</div>

---

> **Status: Solo usable — v0.2.0.** Six MCP tools, `serve` with cookie-auth React
> UI (en/zh-CN, light/dark), Library browse, remember/record forms, file ingest
> (docx/xlsx/pdf text) storing **corpus** with an optional distill step,
> conflict resolve, backup/eval/term-map, store conformance.
> Verify with `make verify-tier3`. See [docs/web.md](docs/web.md) and
> [docs/ingest.md](docs/ingest.md). **PostgreSQL Open + conformance is shipped;
> team auth remains later.**

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

## Solo now, team later

0.x supports **one person on one machine**. That is not a temporary limitation of
the beta — it is the product scope (`design/adr/0007-solo-only-scope.md`), and
everything shipped is built for it: one daemon, one SQLite file, one token.

| | Solo (`sqlite://`) — **supported** | Team — **planned, not implemented** |
|---|---|---|
| Data | one file on your machine | — |
| Identity | one implicit local user | users / teams / tokens do not exist yet |
| Backup | `VACUUM INTO` snapshot | — |
| Started with | `rockymem serve` | — |

`--dsn postgres://…` works today, but it buys a **PostgreSQL-backed single-user
store**, not collaboration: no auth, no membership, no concurrent-writer
validation, no backup. See [docs/postgres.md](docs/postgres.md).

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

- **MCP server** — six tools over **stdio and Streamable HTTP** (`POST /mcp`,
  stateless, Bearer auth). Prefer `make serve` first; MCP/CLI reuse the daemon
  via discovery files next to the DB (same `--dsn`).
- **Two knowledge types** — versioned memory with stable lineages; an experience
  ledger with trust states, conflict detection and falsification.
- **CLI ops** — `review`, `reindex`, `purge`, `export`, `backup`/`make backup`, `restore`, `eval`.
- **Retrieval** — FTS5 trigram + short-query LIKE + zh→en term-map expansion.
- **Web SPA** — search, **library**, **remember** / **record** forms, review, entry,
  **ingest**, overview, settings (one-click backup); en·zh-CN; light/dark (`make web`).
- **Ingest** — two-phase preview/confirm (plain + docx/xlsx/PDF text-layer, redaction
  spans). Stores a document as **corpus** (`source` + `chunk`) by default; `distill`
  is an opt-in checkbox that also proposes one experience.
- **Conflict adjudication** — when two entries claim different fixes for the same
  symptom, a human rules: adopt / split by conditions / reject both / defer /
  reopen. Rulings are append-only with a required reason; unresolved conflicts are
  flagged in every context pack.
- **Corpus retrieval** — ingested passages appear in a capped `## Background`
  section, never displacing lessons; `mem_get` reads a passage (`c_…`) or lists a
  document (`s_…`); `purge --source` removes a document and its chunks.
- **Idempotent writes** — remember/record content dedupe; memory `rev` CAS (MCP exposes `rev`).

**Shipped (PostgreSQL store slice):**

- **Open + migrate + Store** — `--dsn postgres://…` / `postgresql://…` with
  `pg_trgm` search; conformance via `ROCKYMEM_TEST_PG_DSN`.

**Next (in scope):**

- **WU / git timeline, vectors** — see `design/DESIGN.md` §16.2.
- **`bench/`** — the latency numbers are still targets, not measurements.

**Out of scope for 0.x:**

- **Team mode** — shared auth, membership, multi-writer concurrency, PG backup.
  Planned, deliberately unscheduled; see `design/adr/0007-solo-only-scope.md`.
- **MySQL**, hosted SaaS, multi-tenant isolation.

## Architecture at a glance

```
MCP (Cursor, DSH) ─┐
HTTP /api/v1 ──────┼─▶ daemon ─▶ ingest funnel ─▶ store ─▶ SQLite / PostgreSQL
Web UI (SSE) ──────┘             retrieve pipeline ─▶ context pack
```

One daemon owns the database. MCP clients are thin proxies — Cursor and DSH each
spawn their own subprocess, and a subprocess must never open the database
directly. See [ADR-0005](design/adr/0005-technology-stack.md).

## Install

```bash
go install github.com/linyu19872008/rocky-mem/cmd/rockymem@latest
```

Or download a binary from [Releases](https://github.com/linyu19872008/rocky-mem/releases):
darwin / linux / windows, amd64 / arm64, with `checksums.txt`. The binary is
static (CGO disabled) and embeds the web UI, so it needs nothing else.

```bash
rockymem init  --dsn sqlite://$HOME/.local/share/rocky-mem/rocky-mem.db
rockymem serve --dsn sqlite://$HOME/.local/share/rocky-mem/rocky-mem.db
# MCP over stdio:  rockymem mcp
# MCP over HTTP:   http://127.0.0.1:7432/mcp   (bearer token beside the DB)
```

Requires Go 1.26+ **only** for `go install`; the release binaries do not.

## Quick start (this repo)

```bash
make build
./scripts/seed-demo.sh          # demo prefs + global-scoped experiences
make serve                      # exclusive writer on .local/rocky-mem.db
# another terminal / Cursor MCP uses the same DSN and talks HTTP to serve
# bearer in .local/rocky-mem.token is reused across restarts; rotate with:
#   ./bin/rockymem --dsn sqlite://$PWD/.local/rocky-mem.db serve --rotate-token
make verify-tier1
```

Project MCP config: [`.cursor/mcp.json`](.cursor/mcp.json) (same DSN as seed/serve).

```bash
rockymem init --dsn sqlite://~/.local/share/rocky-mem/rocky-mem.db
rockymem serve --dsn sqlite://~/.local/share/rocky-mem/rocky-mem.db
```

## Non-goals

Screen recording and keystroke capture · hosted SaaS · multi-tenant isolation
between untrusted organisations · productivity reporting · translating your notes
· a general-purpose note app · graph visualizations · MySQL support.

The productivity one matters most: a store used for surveillance gets routed
around, and a biased sample is worse than no sample.

## The agent skill

This repository also publishes **`rocky-mem-skill`** (npm): the agent-facing
playbook for when to consult the ledger, how to read a context pack, and which
writes get rejected. It ships from the same tag as the server, and CI asserts
that `skills/rocky-mem/SKILL.md` and `npm/skill/package.json` agree on the
version — the skill can never describe a contract it does not belong to.

| Host | Install |
|---|---|
| DeepSeek Harness | the [`rocky-mem-dsh`](https://github.com/linyu19872008/rocky-mem-dsh) bundle registers the skill and injects context at session start |
| Cursor | `npx rocky-mem-skill install --host cursor` |
| Claude Code | `npx rocky-mem-skill install --host claude-code` |
| Codex / any AGENTS.md host | `npx rocky-mem-skill install --host codex` |

The source of truth is `skills/rocky-mem/`; `npm/skill/` is generated from it by
`prepack` at publish time and is not edited by hand.

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
| [docs/web.md](docs/web.md) | Embedded React SPA |
| [docs/ingest.md](docs/ingest.md) | File ingest + redaction preview |
| [docs/backup.md](docs/backup.md) | Backup / restore / retain |
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
