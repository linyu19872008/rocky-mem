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
DSN="sqlite://$PWD/.local/rocky.db"
./bin/rockymem --dsn "$DSN" search "ConfigMap rollout"
./bin/rockymem --dsn "$DSN" show e_… --raw
./bin/rockymem --dsn "$DSN" feedback e_… --outcome helpful
```

## Cursor MCP

```json
{
  "mcpServers": {
    "rocky-mem": {
      "command": "/absolute/path/to/rocky-mem/bin/rockymem",
      "args": ["--dsn", "sqlite:///absolute/path/to/rocky-mem/.local/rocky.db", "mcp"],
      "env": {}
    }
  }
}
```

Tools: `mem_context`, `mem_search`, `mem_get`, `mem_record`, `mem_remember`, `mem_feedback`.

## Acceptance

1. `mem_record` with evidence returns `status=published` when quality gates pass.
2. `mem_search` returns a pack with `## Applies` / `## Do not use` and uids.
3. `mem_get` returns evidence; `mem_feedback --outcome wrong` demotes trust to `contested`.
4. `rockymem doctor` reports `search_doc_ok: true` and `schema_version: "2"`.

PostgreSQL, Web UI, and team auth remain out of scope.
