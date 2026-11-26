# Security policy

## Reporting

Do not open a public issue for a vulnerability. Use GitHub's private
"Report a vulnerability" advisory on this repository, or the maintainer address
listed in `go.mod` / repository metadata.

Expect an acknowledgement within 7 days. A confirmed issue in a released version
is shipped as a patch release with a GitHub Security Advisory; credit is given
unless you ask otherwise.

## Deployment scope

rocky-mem 0.x has **one supported deployment**: a single user, on one machine,
with loopback-only networking and one bearer token. The storage backend can be
SQLite (default) or PostgreSQL; that choice changes where the bytes live, not who
may reach them.

| | Solo — **supported** | Team — **not implemented** |
|---|---|---|
| Data location | One SQLite file, or one PostgreSQL database you own | — |
| Network surface | Loopback only | — |
| Identity | One implicit local user (`app_user(1,'local')`) | users/teams/memberships: tables exist, **no code enforces them** |
| Trust boundary | The OS user account | — |

**Do not share one instance between people.** There is no authorization layer to
enforce it: every client holding the token can read and write everything, and the
concurrent-write path has never been validated. This is a deliberate scope
decision (`design/adr/0007-solo-only-scope.md`), not an unfinished edge case.

There is no hosted mode and no plan to build one. Multi-tenant isolation is
explicitly out of scope: do not run one instance for mutually untrusted
organisations.

## Threat model

Only solo threats apply. Each row below is backed by shipped code — claims that
could not be backed were removed from this table rather than softened.

| Threat | Relevance | Mitigation |
|---|---|---|
| Another local user reads the store | High on shared machines | State directory mode 700, database 600, token 600 |
| Another local process calls the API | High | Loopback bind + bearer token checked on every request |
| A leaked token is reused | Medium | One token file with mode 600; rotate with `rockymem serve --rotate-token` and reload MCP clients. The token is stored **as a file, not hashed** — it is a local convenience credential, not a server-side secret |
| A prompt-injected agent writes poisoned entries | High | Experience entries require evidence and default to `proposed`; one `wrong` feedback demotes to `contested` immediately |
| A buggy client floods the store | Medium | Quality gate defaults to `proposed`. **There is no rate limiter** |
| Sensitive content persisted by accident | High | Redaction before persist; L4 refused with `ERR_POLICY_REJECTED`; two-phase ingest with a visible diff |
| A dropped file contains a private key | High | Rule packs + sensitivity classification; the preview shows replacements before commit |
| Exported data leaks beyond the intended tier | Medium | `export --level L1` is a whitelist, not a filter |
| Backups are readable | Medium | `VACUUM INTO` targets inherit the state directory mode |
| Nobody can tell who read what | Low | Reads are audited (`action='read'`) for `mem_context`, `mem_search`, `mem_get` |

**Not applicable today:** cross-team access, admin-can-read-everything, token
scopes, revocation across many tokens. All of those are team-mode threats, and
team mode is not implemented (`design/adr/0007-solo-only-scope.md`). Do not
assume any of them are mitigated.

## Deliberate non-goals

- Multi-tenant isolation between mutually untrusted organisations.
- Encryption at rest. Use full-disk encryption or the database server's own
  facility; both do it better than an application layer would.
- Defending against an attacker who already has the user's account.

## Product policy that is security-relevant

Two commitments are enforced in the product, not merely stated:

1. **No per-person productivity reporting.** No UI view, API endpoint or export
   aggregates output by person. This is not modesty: a knowledge store used for
   surveillance gets routed around, and a biased sample is worse than no sample.
2. **Audit reads are themselves audited.** Reading the audit log requires an
   admin role, and that read writes an audit row. An administrator cannot browse
   the log without leaving a trace.

These live here rather than in the README because they are properties users rely
on, and a regression in either is a security regression.

## Scope

**In scope:** token handling, the HTTP and MCP surface, authorization across
scopes and teams, redaction correctness and bypasses, path traversal in file
ingest, SQL injection, dependency vulnerabilities, data exposure through exports,
logs or diagnostic bundles.

**Out of scope:** attacks requiring local code execution as the user, physical
access, misconfigured PostgreSQL instances, and issues in third-party MCP clients.

## Dependencies

Runtime dependencies are few by design. Go's standard library covers HTTP, JSON,
logging, embedding and testing; third-party surface is the SQLite and PostgreSQL
drivers, the MCP SDK, the CLI framework, and `sqlc`-generated (build-time only)
code.

Dependency updates arrive through Dependabot and are reviewed like code. A new
dependency that makes outbound network calls at runtime is rejected on sight: the
tool must not phone home.
