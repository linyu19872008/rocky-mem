# Tier-3 — conformance + richer web

## What shipped

| Item | Surface |
|------|---------|
| Store conformance | `internal/store/conformance` — dialect-neutral suite; SQLite runner green. Postgres plugs the same `Suite` later. |
| Web entry detail | `/entry?uid=m_…\|e_…` — evidence, conflict peers, approve/reject/feedback |
| Search polish | Hit list with links into entry detail |
| Review polish | Links to entry detail; shows trust/type |

Cookie session auth (HttpOnly) remains: open logged URLs, no Bearer typing.

## Verify

```bash
make verify-tier3
```

## Notes

- Conformance asserts *observable* Store behaviour: local user, memory supersede
  + rev conflict, experience search_doc / duplicate find, raw hash lookup,
  feedback + trust, purge by uid, project root coalesce, doctor.
- Do not fork assertions per dialect; document intentional divergence in
  `design/schema.md` instead.
- Daily backups: [backup.md](backup.md) (`make backup`, `scripts/backup-daily.sh`).
