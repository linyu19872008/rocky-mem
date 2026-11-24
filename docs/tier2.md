# Tier-2 — backup, retrieval, minimal web

## What shipped

| Item | Surface |
|------|---------|
| Backup / restore | `rockymem backup [--out path]`, `rockymem restore <snapshot.db>` (`VACUUM INTO`) |
| Short-query LIKE | ≤2-rune queries and empty-FTS fall back to escaped `LIKE` |
| Term-map expansion | Embedded `data/seed/term-map.zh-en.yaml`; zh phrases expand to EN search variants (weight 0.7) |
| Minimal web | With `rockymem serve` (default): `http://127.0.0.1:7432/` search debug, `/review` queue. `--no-web` disables |

## Verify

```bash
make verify-tier2
```

## Ops notes

1. Prefer `make serve` before Cursor MCP.
2. `restore` refuses while discovery files indicate a live serve.
3. Web UI injects the loopback bearer token into the page (solo trust boundary).
