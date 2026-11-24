# Web UI (SPA skeleton)

Solo `rockymem serve` embeds a Vite + React + FormatJS app (ADR-0005).

## Stack

| Piece | Choice |
|---|---|
| App | `web/` — TypeScript, Vite, React Router |
| i18n | `react-intl` + `web/locales/{en,zh-CN}.json` |
| Theme | CSS variables + `data-theme`; System / Light / Dark segmented control |
| Shell | Left sidebar (grouped) + top bar (version, locale, theme, Local badge) |
| Embed | Build output → `internal/web/dist` via `go:embed` |
| Auth | HttpOnly cookie `rocky_mem_session` (same token as Bearer) |

UI strings are localized. **Stored entry bodies are never translated** (see [i18n.md](i18n.md)).

## Commands

```bash
make web      # pnpm install + i18n:check + vite build
make build    # make web && go build
make serve    # rebuild + exclusive writer; open logged URLs
```

Dev against a running serve:

```bash
make serve          # terminal 1
pnpm --dir web dev  # terminal 2 — proxies /api to :7432
```

## Layout

- **Top bar:** brand, `v0.1.x` pill, EN/中文 segmented, theme segmented, Local pill.
- **Side nav:** Work (Search, Library, Review, Entry) · Capture (Ingest) · System (Overview, Settings).
- **Page body:** `PageHeader` (title + actions) → optional `StatStrip` / `FilterChips` → content cards.

## Routes

| Path | Status |
|---|---|
| `/` | Search (retrieval debugger); empty state links to Library |
| `/library` | Browse memories + experiences via `POST /api/v1/browse` (no query required) |
| `/remember` | Write a standing memory (`POST /api/v1/remember`) |
| `/record` | Write an experience with evidence (`POST /api/v1/record`) |
| `/review` | Review queue with KPI + counted filter chips |
| `/entry` | Entry detail + conflict compare; approve/reject in page header |
| `/settings` | Preferences / doctor / **one-click backup** (`POST /api/v1/backup`) |
| `/overview` | KPIs (clickable → Library filters) + recent library + pending review |
| `/ingest` | Two-phase ingest: drop/paste → **extract** (plain/docx/xlsx/pdf text) → redact preview → confirm |

## Locale resolution

1. `?lang=`
2. Cookie `rocky_mem_locale`
3. `Accept-Language`
4. `en`

Server injects `window.__ROCKY_BOOT__={locale}` into `index.html` so the first paint matches.

## Theme

`localStorage.rocky_mem_theme` = `system` | `light` | `dark`. A tiny inline script in `index.html` applies the resolved theme before React mounts (no flash).
