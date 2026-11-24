# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Until `1.0.0`, minor versions may include breaking changes. Breaking changes are
always marked `!` in the commit subject and listed under **Changed** with a
migration note.

## [Unreleased]

### Changed

- **Stable serve bearer:** mint `*.token` only when missing/empty; shutdown keeps
  the token file. Use `rockymem serve --rotate-token` (or delete the file) to
  rotate; reload Cursor MCP afterwards. See `docs/tier1.md`.

### Added

- **Two-phase ingest:** `POST /api/v1/ingest/preview|confirm` + Web `/ingest`
  (redaction diff → proposed experience).
- **Search hit cards:** `PackResult.hits` with title / applies / invalid; Web
  Search page cards.
- **Conflict resolve:** `POST /api/v1/review/resolve` (`keep` | `archive_all`)
  on Entry when contested peers exist.
- **Eval thickening:** more corpus entries + zh/en queries; eval uses term-map
  via `NewService` (14/14 recall@10).
- **Backup LaunchAgent installer:** `scripts/install-backup-launchd.sh`.
- **Web SPA skeleton:** Vite + React + FormatJS embedded in `serve` (`make web`);
  en/zh-CN catalogs, light/dark/system theme, shared component kit; sidebar +
  top-bar admin shell (PageHeader / StatStrip / FilterChips); routes for
  search/review/entry/settings/overview/ingest. See `docs/web.md`.
- **Tier-3 (local):** `internal/store/conformance` SQLite suite (supersede rev
  conflict, raw hash, feedback/trust, purge); web `/entry` detail with
  side-by-side conflict compare + feedback; `/review` trust filters and conflict
  grouping; `/settings` doctor snapshot; HttpOnly cookie web auth (no Bearer in
  page). See `docs/tier3.md`.
- **Backup automation:** `backup --dir/--retain`, `make backup`,
  `scripts/backup-daily.sh`, docs in `docs/backup.md`.
- **Tier-2:** `backup`/`restore` (`VACUUM INTO`), short-query LIKE hardening,
  embedded term-map query expansion, minimal search/review web UI on `serve`
  (`--no-web` to disable). See `docs/tier2.md` and `make verify-tier2`.
- **Tier-1 operational slice:** `rockymem serve` (flock + discovery + bearer
  `/api/v1`), MCP/CLI daemon fallback, `review` / `reindex` / `purge` / `export` /
  `eval`, remember/record idempotency + rev CAS, retrieval corpus with
  recall@10 ≥ 0.8. See `docs/tier1.md` and `make verify-tier1`.
- **Project identity coalesce:** same `root_hint` merges `local/…` and remote
  project rows (remap `scope_ref` + `raw.project_id`). Demo seed uses
  `--scope global`. `make serve` / `scripts/serve-local.sh` is the default
  multi-window writer path.
- **P0 SQLite slice (runnable):** Go binary `rockymem` with SQLite store,
  CLI (`init`, `mcp`, `search`, `show`, `remember`, `record`, `feedback`, `doctor`),
  and all six MCP tools (`mem_context` / `mem_search` / `mem_get` / `mem_record` /
  `mem_remember` / `mem_feedback`). Schema v2 adds `feedback`. See
  `docs/quickstart-p0a.md`. PostgreSQL and Web UI are not in this slice.
- Project Cursor MCP config (`.cursor/mcp.json`) and `make mcp-smoke` stdio
  client that exercises all six tools.
- **Tier-0 adoption gate:** `AGENTS.md`, `.cursor/rules/rocky-mem-mcp.mdc`,
  `rockymem adopt-verify`, and `make verify-tier0` (protocol + workflow + audit
  verdict with unit tests). See `docs/tier0-adoption.md`.
- Design set: `design/DESIGN.md`, `design/schema.md`, `design/README.md`, and ADRs 0001–0006.
- Dual-dialect schema: SQLite (FTS5 `trigram`) and PostgreSQL (`pg_trgm` GIN),
  with a materialised `search_doc` table replacing trigger-maintained indexes.
- Identity and tenancy seams from P0: `app_user`, `team`, `membership`,
  `api_token`, `project`.
- Optimistic locking (`rev`) on all mutable content, plus an `idempotency` table.
- Contract documents: `docs/mcp-tools.md`, `docs/error-codes.md`.
- Localization specification and glossary: `docs/i18n.md`, `docs/i18n-glossary.md`.
- Seed bilingual term map (`data/seed/term-map.zh-en.yaml`) for query expansion.
- Project governance scaffolding: Apache-2.0 license, bilingual README,
  `CONTRIBUTING.md`, `SECURITY.md`.

### Changed

- **Documentation layout**: design documents now live in `design/` (Chinese) and
  contracts/specs in `docs/` (English). The split is by audience; see
  `design/README.md` and `docs/README.md`.
- **Language and stack** (`ADR-0005`): Go single binary replaces the earlier
  Python plan. Solo (`sqlite://`) and team (`postgres://`) modes share one binary.
- **Storage strategy** (`ADR-0006`): SQLite and PostgreSQL only. MySQL is not
  supported; the reasons are recorded, not implied.
- `scope_ref` for project scope now stores a `project_id` instead of a local
  path, so two clones of the same repository resolve to one project.
- Design documents are now written in **Chinese**; contracts and engineering
  documents remain English. See `design/README.md` for the split.
- The "standard library only" runtime constraint is withdrawn: few, boring,
  auditable dependencies are the rule.

### Deprecated

- `ADR-0003` (SQLite only, no concurrent writers). Superseded by `ADR-0005`; the
  file is kept as history.

### Removed

- Database triggers and FTS5 external-content tables. Retrieval documents are now
  written by a single application-layer function; `reindex` rebuilds them.

[Unreleased]: https://github.com/<org>/rocky-mem/commits/main
