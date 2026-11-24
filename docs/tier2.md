# Tier-2 — backup, retrieval, minimal web

## What shipped

| Item | Surface |
|------|---------|
| Backup / restore | `rockymem backup [--out path\|--dir DIR --retain N]`, `make backup`, `scripts/backup-daily.sh`, `rockymem restore <snapshot.db>` (`VACUUM INTO`) — see [backup.md](backup.md) |
| Short-query LIKE | ≤2-rune queries and empty-FTS fall back to escaped `LIKE` |
| Term-map expansion | Embedded `data/seed/term-map.zh-en.yaml`; zh phrases expand to EN search variants (weight 0.7) |
| Minimal web | With `rockymem serve` (default): open logged URLs in a browser — HttpOnly cookie auth, no Bearer typing. `--no-web` disables |

## Verify

```bash
make verify-tier2
```

## Ops notes

1. Prefer `make serve` before Cursor MCP.
2. `restore` refuses while discovery files indicate a live serve.
3. Web UI uses an HttpOnly session cookie (solo loopback); MCP/CLI still use the token file.
