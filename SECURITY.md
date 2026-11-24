# Security policy

## Reporting

Do not open a public issue for a vulnerability. Use GitHub's private
"Report a vulnerability" advisory on this repository, or the maintainer address
listed in `go.mod` / repository metadata.

Expect an acknowledgement within 7 days. A confirmed issue in a released version
is shipped as a patch release with a GitHub Security Advisory; credit is given
unless you ask otherwise.

## Deployment modes and what they imply

rocky-mem runs in one of two modes, selected by the DSN. The threat model differs
enough that they are described separately.

| | Solo (`sqlite://`) | Team (`postgres://`) |
|---|---|---|
| Data location | One file on the user's machine | A PostgreSQL instance the operator runs |
| Network surface | Loopback only | Configurable bind address |
| Identity | One implicit local user | Tokens, users, teams, memberships |
| Trust boundary | The OS user account | The server, plus every client holding a token |
| Operator | The user | Whoever deployed it |

There is no hosted mode and no plan to build one. Multi-tenant isolation is
explicitly out of scope: do not run one instance for mutually untrusted
organisations.

## Threat model

| Threat | Mode | Relevance | Mitigation |
|---|---|---|---|
| Another local user reads the store | Solo | High on shared machines | State directory mode 700, database 600, token 600 |
| Another local process calls the API | Solo | High | Loopback bind + bearer token required on every request |
| A leaked token is used by someone else | Team | High | Tokens are stored as Argon2id hashes, scoped, expirable, revocable; `last_used_at` is surfaced in the UI |
| One team reads another team's entries | Team | High | Scope inference only narrows; `project_id` and team membership are validated in the store layer, not just in the handler |
| A prompt-injected agent writes poisoned entries | Both | High | Experience entries require evidence and default to `proposed`; one `wrong` feedback demotes immediately |
| A malicious or buggy MCP client floods the store | Both | Medium | Quality gate defaults to `proposed`; per-token write rate limit |
| Sensitive content persisted by accident | Both | High | Redaction before persist; L4 rejected at ingest; two-phase ingest with a visible diff for file drops |
| A dropped file contains a private key | Both | High | Redaction rule packs + sensitivity classification; the preview shows replacements before commit |
| Exported data leaks beyond the intended tier | Both | Medium | `export --level L1` is a whitelist, not a filter |
| Backups are world-readable | Both | Medium | `VACUUM INTO` targets inherit directory mode; team-mode backups are the operator's responsibility and are documented as such |
| An administrator reads everything | Team | Inherent | This cannot be prevented technically. It is documented, and the audit log records who read what, including administrator reads |

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
