# Tier-1 — daemon, maintenance, eval

Gate for the first operational slice after adoption (Tier-0).

## What shipped

| Item | Command / surface |
|------|-------------------|
| Solo daemon | `rockymem serve [--bind 127.0.0.1:7432]` — flock + discovery files + bearer `/api/v1` |
| Thin MCP/CLI | Prefer healthy daemon (`ROCKYMEM_URL` / `*.addr`+`*.token`); else local SQLite under flock |
| Review | `rockymem review list\|approve\|reject` |
| Maintain | `rockymem reindex`, `purge`, `export` |
| Idempotency | `mem_remember` content-hash → `unchanged`; `mem_record` title+symptom+fix duplicate → `unchanged`; remember `rev` CAS |
| Eval | `rockymem eval` / `go test ./internal/eval` — recall@10 ≥ 0.8 on fixed corpus |

## Process model

1. Start `rockymem serve` (owns the DB).
2. Cursor MCP / CLI talk HTTP to the daemon.
3. If serve is down, CLI/MCP open the DB directly and take the lock (exclusive).

Discovery files sit next to the DB: `rocky-mem.lock`, `rocky-mem.addr`, `rocky-mem.token`.

**Token lifecycle (Solo):** mint only when `*.token` is missing or empty; otherwise
reuse. Shutdown clears `*.addr` only (keeps the token). Rotate explicitly:

```bash
rockymem serve --rotate-token   # or: rm .local/rocky-mem.token && make serve
```

After rotate, reload Cursor MCP (old long-lived clients still hold the previous bearer).

## Verify

```bash
make verify-tier1
```

Runs unit tests (core idempotency/CAS, daemon lock, API client, eval recall@10) and `go build`.
