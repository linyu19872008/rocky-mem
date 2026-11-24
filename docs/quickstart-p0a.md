# P0 quickstart (SQLite slice)

Proves: **an agent can record, retrieve, inspect, and correct a lesson with provenance**.

## Build

```bash
make build
./bin/rockymem version   # 0.1.0-p0b
```

## Initialise and seed

```bash
export ROCKYMEM_HOME="$PWD/.local"
./bin/rockymem init
./scripts/seed-demo.sh
DSN="sqlite://$PWD/.local/rocky-mem.db"
./bin/rockymem --dsn "$DSN" search "ConfigMap rollout"
./bin/rockymem --dsn "$DSN" show e_… --raw
./bin/rockymem --dsn "$DSN" feedback e_… --outcome helpful
```

## Cursor MCP

Project config is already at [`.cursor/mcp.json`](../.cursor/mcp.json) (uses `${workspaceFolder}`).

1. `make build` and ensure `.local/rocky-mem.db` exists (`./scripts/seed-demo.sh`).
2. Reload MCP: **Cursor Settings → MCP** → enable/refresh `rocky-mem`, or restart Cursor.
3. Confirm tools appear: `mem_context`, `mem_search`, `mem_get`, `mem_record`, `mem_remember`, `mem_feedback`.
4. Protocol smoke (no Cursor UI required): `make mcp-smoke`.

Manual absolute-path equivalent:

```json
{
  "mcpServers": {
    "rocky-mem": {
      "type": "stdio",
      "command": "/absolute/path/to/rocky-mem/bin/rockymem",
      "args": ["--dsn", "sqlite:///absolute/path/to/rocky-mem/.local/rocky-mem.db", "mcp"]
    }
  }
}
```

## Acceptance

1. `mem_record` with evidence returns `status=published` when quality gates pass.
2. `mem_search` returns a pack with `## Applies` / `## Do not use` and uids.
3. `mem_get` returns evidence; `mem_feedback --outcome wrong` demotes trust to `contested`.
4. `rockymem doctor` reports `search_doc_ok: true` and `schema_version: "2"`.

PostgreSQL, Web UI, and team auth remain out of scope.
