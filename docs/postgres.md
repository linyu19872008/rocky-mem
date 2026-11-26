# PostgreSQL store (single-user alternative)

SQLite remains the default. This backend exists so that one person who prefers
PostgreSQL can keep their store there: `--dsn postgres://…` /
`postgresql://…` implements the same `store.Store` interface and passes the same
`store/conformance.Suite`.

**This is not team mode.** rocky-mem 0.x is solo-only (`design/adr/0007-solo-only-scope.md`).
There is no auth, no `membership` enforcement, no multi-writer validation, and no
backup/restore for this backend. Running one instance for several people is not
supported and will not be made to work by configuration.

## Requirements

- PostgreSQL 14+ with the `pg_trgm` extension (migration runs
  `CREATE EXTENSION IF NOT EXISTS pg_trgm`).
- DSN form: `postgres://user:pass@host:5432/dbname?sslmode=disable`

## Run

```bash
# serve (discovery under $ROCKYMEM_HOME/pg-<sha8>/; no flock)
./bin/rockymem --dsn 'postgres://rocky:rocky@127.0.0.1:5432/rocky?sslmode=disable' serve

# conformance
export ROCKYMEM_TEST_PG_DSN='postgres://rocky:rocky@127.0.0.1:5432/rocky?sslmode=disable'
make verify-postgres
```

Ephemeral Docker example:

```bash
docker run -d --name rocky-pg \
  -e POSTGRES_PASSWORD=rocky -e POSTGRES_USER=rocky -e POSTGRES_DB=rocky \
  -p 55432:5432 postgres:16-alpine
docker exec rocky-pg psql -U rocky -c 'CREATE EXTENSION IF NOT EXISTS pg_trgm'
ROCKYMEM_TEST_PG_DSN='postgres://rocky:rocky@127.0.0.1:55432/rocky?sslmode=disable' \
  make verify-postgres
```

## Gaps and non-goals

**Out of scope by decision** (`ADR-0007`) — do not expect these in 0.x:

- Team auth, `membership` enforcement, multiple API tokens.
- Multi-writer concurrency: no advisory lock, no idempotency wiring, and the
  concurrent path has never been load-tested.
- Backup/restore for this backend (`VACUUM INTO` is SQLite-only; use `pg_dump`).

**Ordinary limitations of this slice:**

- 2-character Chinese queries fall back to `LIKE` and do not use the trigram index
  at scale; `zhparser` is not wired.
- Discovery paths are hashed under `$ROCKYMEM_HOME/pg-<sha8>/`; there is no flock,
  so two daemons against the same PG database are not coordinated.
