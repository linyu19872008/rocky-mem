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
- **Side nav:** Work (Search, Review, Entry) · Capture (Ingest) · System (Overview, Settings).
- **Page body:** `PageHeader` (title + actions) → optional `StatStrip` / `FilterChips` → content cards.

## Routes

| Path | Status |
|---|---|
| `/` | Search |
| `/review` | Review queue with KPI + counted filter chips |
| `/entry` | Entry detail + conflict compare; approve/reject in page header |
| `/settings` | Preferences / doctor / ops cards |
| `/overview` | Real KPIs via `GET /api/v1/stats` + recent pending list |
| `/ingest` | Drop-zone placeholder for later two-phase ingest |

## Locale resolution

1. `?lang=`
2. Cookie `rocky_mem_locale`
3. `Accept-Language`
4. `en`

Server injects `window.__ROCKY_BOOT__={locale}` into `index.html` so the first paint matches.

## Theme

`localStorage.rocky_mem_theme` = `system` | `light` | `dark`. A tiny inline script in `index.html` applies the resolved theme before React mounts (no flash).
