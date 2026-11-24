# Documentation

This directory holds **contracts and specifications** — documents whose readers
are outside the maintainer group: contributors, third-party clients, and agent
integrators.

Design documents (goals, architecture, trade-offs, ADRs) live in
[`../design/`](../design/) and are written in Chinese. The split is by audience,
not by importance; see [`../design/README.md`](../design/README.md).

## Contents

| File | Purpose | Stability |
|---|---|---|
| [mcp-tools.md](mcp-tools.md) | The six agent-facing MCP tools: JSON Schema, outputs, evidence resolution, error mapping | Stable, semver'd |
| [error-codes.md](error-codes.md) | The stable error registry. Codes are never renamed or reused | Stable, part of the public API |
| [i18n.md](i18n.md) | Locale resolution, message catalogs, ICU rules, multilingual content and retrieval | Normative |
| [i18n-glossary.md](i18n-glossary.md) | Canonical term translations, plus the alternatives that were rejected and why | Normative for translators |
| [tier0-adoption.md](tier0-adoption.md) | Agent adoption protocol and live checklist | Process |
| [tier1.md](tier1.md) | Daemon, maintain CLI, idempotency, retrieval eval gate | Process |
| [tier2.md](tier2.md) | Backup, term-map, LIKE, minimal web | Process |
| [tier3.md](tier3.md) | Store conformance + richer web entry/review | Process |
| [quickstart-p0a.md](quickstart-p0a.md) | Runnable P0 SQLite slice quickstart | Guide |

## Rules for this directory

1. **English only.** These files are read by people who do not read Chinese.
2. **Identifiers are literal.** Tool names, field names, enum values and error
   codes appear here exactly as they appear in code. Never translate them, and
   never paraphrase a parameter name.
3. **Contracts change in lockstep with code.** A pull request that changes a tool
   signature or adds an error code updates these files in the same commit.
4. **Stability is explicit.** `mcp-tools.md` and `error-codes.md` are public API.
   `i18n.md` is normative for implementers but not a compatibility promise.

## Where things go

```
design/     why and what — Chinese, for maintainers
docs/       contracts and specs — English, for integrators
README.md   the front door — English, with a Chinese translation
```

If you are unsure which side a document belongs to, apply the test: *would an
outside contributor need this to integrate, or would only a maintainer need it to
decide?* Integration material goes here; decision material goes to `design/`.
