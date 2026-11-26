# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Until `1.0.0`, minor versions may include breaking changes. Breaking changes are
always marked `!` in the commit subject and listed under **Changed** with a
migration note.

## [0.2.1] — 2026-11-27

### Added

- **Release binaries.** GoReleaser builds static binaries for
  darwin/linux/windows × amd64/arm64 with `checksums.txt`, triggered by a tag.
- Governance: code of conduct, CODEOWNERS, dependabot, issue forms, and a PR
  template that asks about contract impact.
- CI now runs on Linux **and macOS**, uses `-race` on Linux, checks `gofmt`, and
  runs the web `typecheck` and `i18n:check` that Vite never enforced.

### Changed

- **Module path** is `github.com/linyu19872008/rocky-mem`, matching the
  repository, so `go install` and third-party imports resolve.
- README gained badges and an Install section.

## [0.2.0] — 2026-11-27

**This release supersedes the unreleased 0.1.3 and 0.1.4 notes below** — those
two sections describe work that ships here for the first time.

### Added

- **MCP over Streamable HTTP.** `POST /mcp` serves the same tools as stdio,
  stateless, behind the existing bearer token. `GET`/`DELETE` answer 405 by
  design; protocol revisions 2024-11-05 through 2026-07-28 are negotiated by
  the SDK.
- **`doctor` publishes capability.** `GET /api/v1/doctor` now carries
  `contract`, `version`, `features[]`, and an `mcp` block, so clients decide
  what they may use instead of inferring it from the release string.
- **Portable skill** at `skills/rocky-mem/` (playbook, references, host
  adapters) published as `rocky-mem-skill` from this repository.
  `make verify-skill` asserts the skill, the npm package, `tools.json`, and the
  real error-code registry agree.
- `rocky-mem-dsh` bundle (separate repository) consumes that contract.

### Changed

- `docs/mcp-tools.md` documents both transports instead of promising a
  `POST /mcp` that did not exist.

## [0.1.4] — 2026-11-27

### Added

- **Conflict adjudication** (`ADR-0009`, migration `0004`): `conflict_resolution`
  (append-only rulings) and `link` (relations). Five actions: `adopt`, `split`,
  `reject_both`, `defer`, `reopen`. A reason is required; `split` requires four
  non-empty, non-identical conditions and writes them back to the entries.
- **`mem_get` exposes `resolutions`** for an experience's conflict group, oldest
  first, so "why is this active" is answerable.
- **Context packs flag unresolved conflicts** with a `> Contested` line and the
  peer uid; members of one group are rendered adjacent.
- **CLI**: `rockymem review resolve <uid> --action … --note …`.

### Changed

- **`keep` / `archive_all` are gone** (`!` for HTTP clients). `action:"adopt"`
  replaces `keep`, and the losing side is now `falsified` (demonstrably wrong)
  rather than `archived` (out of circulation). Callers must pass `note`.
- **Archiving now drops the search document.** `review reject` previously left the
  index row behind, which made `doctor` report drift; `setTrust` now owns the
  index lifecycle.

## [0.1.3] — 2026-11-26

### Added

- **Corpus layer** (`ADR-0008`, migration `0003`): `source` + `chunk` tables,
  `mem_get` for `s_…`/`c_…`, `purge --source`, chunk-aware `reindex` / `doctor`.
  Ingest now stores a document as background material by default; `distill` is an
  opt-in checkbox that also proposes one experience.
- **Corpus retrieval**: passages are returned in a capped `## Background` section
  (2 items / 1 per document by default, 5 / 2 when `kinds` asks for `chunk`) and
  only while ≥30 % of the budget remains.
- **Re-ingest versioning**: the same filename with different content archives the
  previous source and drops its chunks from the index.
- **PostgreSQL store:** `postgres://` / `postgresql://` DSN opens a real
  `pg_trgm` backend (`db/migrations/postgres`, `internal/store/postgres`) that
  shares `store/conformance.Suite` (set `ROCKYMEM_TEST_PG_DSN` to run).
  Backup/restore remain SQLite-only. This is a **single-user alternative
  backend**, not a shared server — see ADR-0007.

### Changed

- **Ingest default changed** from "one proposed experience with truncated body" to
  "corpus + chunks". Callers that relied on the old behaviour pass `mode: "distill"`.
- **Scope narrowed to solo.** `ADR-0007` records that 0.x ships single-user only:
  no auth wiring, `membership`/`team`/`api_token` stay reserved-and-unread,
  `scope=team` is inert, and PostgreSQL is not a collaboration story. Design,
  README (en/zh) and SECURITY now say this instead of implying two usable modes.
- **Error-code registry realigned to `internal/errs/errs.go`.** 14 codes that
  never existed were removed (`ERR_ENTRY_CONFLICT`, `ERR_DB_BUSY`, …), 8 real
  ones were documented (`ERR_NOT_FOUND`, `ERR_POLICY_REJECTED`, `ERR_DB`,
  `ERR_SYS_INTERNAL`, …), and the claim that `ERR_WRITE_CONFLICT` is team-only
  was corrected — it is reachable in solo via a stale `rev`.
- **SECURITY threat model rewritten** to contain only mitigations that exist;
  the Argon2id-hashing and rate-limit claims were false and are gone.
- **Design set calibrated to v0.1.2.** `design/DESIGN.md` §6/§11/§13/§14 now
  describe the shipped code, §15.1 lists every known design-vs-implementation
  gap, and §16 records delivery by Tier instead of plan.
  `design/schema.md` gained the `0002_feedback` migration and the reserved
  `chunk` note; `docs/mcp-tools.md` documents the `cwd` argument on
  `mem_record`/`mem_remember`, the `501` for `kinds: ["chunk"]`, and the inert
  `team` scope.

## [0.1.2] — 2026-11-24

### Added

- **Record form:** Web `/record` for hand-written experiences with quote evidence.
- **Ingest success CTAs:** clearer next steps to Review / Library / Entry.
- **Shallow evidence resolve:** `commit` via `git log -1`, `path` via file stat
  under `cwd` (docs/mcp-tools.md updated).
- **Eval / term-map thickening:** more corpus entries + zh/en short queries;
  extra seed terms (知识库/投喂/脱敏/…).

### Changed

- Version unified to **0.1.2** (CLI / MCP / Web pill).
- README shipped list includes `/record`.

## [0.1.1] — 2026-11-24

### Added

- **Library browse:** `POST /api/v1/browse` + Web `/library` (kind/trust filters;
  no search query required). Overview stats drill into Library.
- **Remember form:** Web `/remember` writes standing memories via `/api/v1/remember`.
- **Settings one-click backup:** `POST /api/v1/backup` with empty body uses
  `$ROCKYMEM_HOME/backups` + retain 14.
- **MCP `mem_remember.rev`:** optimistic-lock arg wired through to core.
- **`ERR_NOT_IMPLEMENTED`:** search `kinds: ["chunk"|"source"]` returns 501.

### Changed

- Version unified to **0.1.1** (CLI / MCP / Web pill).
- README shipped/planned lists updated (ingest + library are not “later”).
- **Stable serve bearer:** mint `*.token` only when missing/empty; shutdown keeps
  the token file. Use `rockymem serve --rotate-token` (or delete the file) to
  rotate; reload Cursor MCP afterwards. See `docs/tier1.md`.

### Added (earlier unreleased → included in 0.1.1)

- **Two-phase ingest:** `POST /api/v1/ingest/preview|confirm` + Web `/ingest`
  (extract → redaction diff → proposed experience). Supports plain text,
  **docx / xlsx / PDF text-layer** via `content_base64`; rejects binary-as-text
  and scanned PDFs. See `docs/ingest.md`.
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
