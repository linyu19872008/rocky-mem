# Contributing

Thanks for considering it. This document is short on purpose: the rules that
matter are listed, and the rest is judgement.

## Language

| Artifact | Language |
|---|---|
| Code, comments, docstrings, commit messages, branch names, PR descriptions | **English** |
| `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md` | English |
| `docs/mcp-tools.md`, `docs/error-codes.md`, `docs/i18n.md` (contracts and specs) | English |
| `design/DESIGN.md`, `design/schema.md`, `design/adr/**`, `design/README.md` (design) | Chinese |
| `README.zh-CN.md` | Chinese |

The split is by audience, not by importance; see `design/README.md` and
[ADR-0004](design/adr/0004-i18n-ui-scope-vs-content-language.md) §7.
Identifiers — tool names, field names, error codes, enum values — are English in
every document and every locale. A translated identifier is a bug.

If English is not your first language, write it imperfectly. An awkward sentence
is easier to fix than a missing contribution.

## Before you start

| Change type | What is required |
|---|---|
| Bug fix, small improvement | Open a PR |
| New tool, new table, changed write semantics, changed retrieval ordering, new dependency | Open an issue first, then an ADR |
| Schema or query change | **Both dialects, in the same PR** (see below) |
| New UI locale | Follow `docs/i18n.md` §10 |
| New term in the term map | One entry in `data/seed/term-map.zh-en.yaml` plus a test with a real query |

Architectural decisions get an ADR **before** the code. An ADR that lists only
benefits will be sent back. See `design/adr/0001-record-architecture-decisions.md`.

## Development setup

Requirements: Go (latest two minor releases). PostgreSQL tooling, `pnpm`, and
sqlc arrive with later phases; P0a is SQLite-only.

```bash
git clone https://github.com/<org>/rocky-mem
cd rocky-mem
go mod download   # if proxy.golang.org times out: GOPROXY=https://goproxy.cn,direct
```

Common commands (P0a):

```bash
make build            # go build -o bin/rockymem ./cmd/rockymem
make test             # go test ./...
make lint             # go vet ./...
./scripts/seed-demo.sh
./bin/rockymem doctor
```

## The two-dialect rule

This is the contribution rule most likely to trip you up.

Any change to `db/queries/**` or `db/migrations/**` must land in **both**
`sqlite/` and `postgres/` in the same PR, and must keep `internal/store/conformance`
green against both backends.

If a feature cannot be expressed identically:

1. Implement it per dialect.
2. Document the divergence in `design/schema.md` §4.
3. Add a conformance test that pins the *observable* behaviour, so the divergence
   is deliberate and tested rather than accidental.

Do not add a JSON column to avoid the divergence. Do not introduce an ORM.

## Commits

[Conventional Commits](https://www.conventionalcommits.org/), English, imperative
mood, scope optional:

```
feat(mcp): add mem_feedback tool
fix(search): fall back to LIKE for two-character queries
feat(store)!: require rev on memory updates
```

`!` marks a breaking change and must be explained in the commit body.

**Sign your commits** (DCO). This project uses the Developer Certificate of Origin
rather than a CLA:

```bash
git commit -s -m "feat: ..."
```

If you cannot certify the DCO, say so in the PR and we will work it out before
merging rather than after.

## Code expectations

- Go idioms, no framework. `net/http` and the standard library first. A new
  dependency needs a paragraph in the PR explaining why the standard library
  cannot do it.
- No ORM, no query builder. SQL lives in `db/queries/<dialect>/*.sql` and goes
  through `sqlc`.
- Errors carry a stable code from `docs/error-codes.md`; adding a code means
  adding it to that registry in the same PR.
- Tests for behavioural changes. A schema change needs a migration for both
  dialects and a conformance test.
- `go test -race` must pass. The daemon serves concurrent readers; writes stay
  single-writer. Do not introduce a second writer path.
- No `SELECT *` in hot paths. No `ORDER BY title` (see `design/schema.md` §9).
- No new network calls. Ever. A PR that adds an outbound request will be closed.

## UI changes

- No hard-coded user-visible strings. Every string goes through the catalog;
  `eslint-plugin-formatjs` enforces this.
- Key grammar is `<domain>.<component>.<element>`; keys are identifiers, never
  English sentences.
- Add or update `web/locales/zh-CN.json` in the same PR. Missing values fall back
  to English; missing keys fail CI.
- Check the layout with the `en-XA` pseudo-locale, which inflates strings by 40 %.
- Keyboard operation is a requirement: the review queue is designed to be used
  without a mouse.
- Use CSS logical properties (`margin-inline-start`), not `margin-left`.
- The frontend build output is embedded in the binary and is **not** committed.

## Documentation changes

- Documentation is part of the change, not a follow-up.
- `design/DESIGN.md` and `design/schema.md` describe the current design. If your PR changes
  the design, update them in the same PR — in Chinese, matching the existing text.
- Contract documents (`docs/mcp-tools.md`, `docs/error-codes.md`) stay English.
- Update `CHANGELOG.md` under `## [Unreleased]` in the appropriate section.

## Pull request checklist

- [ ] Commits are signed off (DCO) and follow Conventional Commits
- [ ] `golangci-lint`, `go test -race ./...` pass locally
- [ ] Schema/query changes updated **both** dialect directories
- [ ] Conformance suite passes for SQLite and PostgreSQL
- [ ] New behaviour has tests; new errors are in the registry
- [ ] UI strings are in the catalog, both locales updated
- [ ] `design/DESIGN.md` / `docs/*.md` updated if behaviour or contracts changed
- [ ] An ADR exists for any decision that would be expensive to reverse
- [ ] `CHANGELOG.md` updated under `[Unreleased]`
- [ ] No new outbound network calls
- [ ] No real personal data, credentials or internal hostnames in fixtures

## Review

Two things reviewers will push on:

1. **What does this cost?** Every feature has a maintenance cost. "It is only 40
   lines" is not an argument; those 40 lines get read every time someone touches
   the file. With two dialects, that cost is often doubled.
2. **What happens when it fails?** Silent degradation is treated as a bug. If a
   feature can fail, it must be visible in `doctor` and in the UI.

Being direct in review is normal here; it is about the code.

## Reporting security issues

Do not open a public issue. See [SECURITY.md](SECURITY.md) for the private
reporting channel.
