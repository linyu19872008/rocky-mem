# Quickstart (SQLite solo + Tier-1 daemon)

Proves: **an agent can record, retrieve, inspect, and correct a lesson with provenance**,
with one exclusive writer when multiple Cursor windows are open.

## Build

```bash
make build
./bin/rockymem version   # 0.1.0-tier1
```

## Initialise and seed

```bash
export ROCKYMEM_HOME="$PWD/.local"
./scripts/seed-demo.sh
DSN="sqlite://$PWD/.local/rocky-mem.db"
./bin/rockymem --dsn "$DSN" search "ConfigMap rollout" --cwd "$PWD"
```

Demo experiences are written with `--scope global` so they stay visible regardless of
how `cwd` resolves `project_id`. Project rows that share the same git root are also
coalesced on upsert (local/… → github.com/…).

## Prefer `serve` (multi-window)

```bash
make serve
# discovery files appear next to the DB:
#   .local/rocky-mem.addr
#   .local/rocky-mem.token
#   .local/rocky-mem.lock
```

MCP and CLI with the same `--dsn` detect a healthy daemon and proxy over HTTP; they
do **not** open SQLite directly while serve holds the lock.

## Cursor MCP

Project config: [`.cursor/mcp.json`](../.cursor/mcp.json) (same DSN as seed/serve).

1. `make build` and `./scripts/seed-demo.sh`.
2. `make serve` (leave it running).
3. Reload MCP: **Cursor Settings → MCP** → enable/refresh `rocky-mem`.
4. Tools: `mem_context`, `mem_search`, `mem_get`, `mem_record`, `mem_remember`, `mem_feedback`.
5. Optional smoke without Cursor UI: `make mcp-smoke`.

## Ops CLI

```bash
./bin/rockymem --dsn "$DSN" review list
./bin/rockymem --dsn "$DSN" reindex
./bin/rockymem --dsn "$DSN" export --format md
./bin/rockymem --dsn "$DSN" eval --min-recall 0.8
make verify-tier1
```

## Acceptance

1. `mem_record` with evidence returns `status=published` when quality gates pass.
2. `mem_search` returns a pack with `## Applies` / `## Do not use` and uids.
3. `mem_get` returns evidence; `mem_feedback --outcome wrong` demotes trust to `contested`.
4. With `make serve` up, a second `rockymem --dsn … mcp` / CLI call does not take the DB lock.
5. `make verify-tier1` exits 0 (see `docs/tier1.md`).

PostgreSQL, Web UI, and team auth remain out of scope.
