# Error codes

A stable public contract. Clients branch on `code`; humans read `message`.

## 1. Conventions

- Format: `ERR_<DOMAIN>_<CONDITION>`, `SCREAMING_SNAKE_CASE`, ASCII.
- Domains: `AUTH`, `REQ`, `ENTRY`, `EVIDENCE`, `SCOPE`, `POLICY`, `DB`, `FTS`, `SYS`.
- Codes are **never renamed, never reused, never silently repurposed.** A code
  that stops being emitted stays in this table, marked deprecated, for at least
  one major version.
- `message` is English, human-readable, and may be reworded between releases —
  which is exactly why clients must not match on it.
- The web UI translates by convention: message key `error.<CODE>`. A missing key
  falls back to the raw `code`, never to a blank string.

## 2. Envelope

HTTP:

```json
{
  "error": {
    "code": "ERR_EVIDENCE_REQUIRED",
    "message": "At least 1 evidence item is required for an experience entry.",
    "params": { "min": 1 },
    "request_id": "req_01J9M…"
  }
}
```

MCP: the same object, JSON-encoded, inside a text content block, with
`isError: true`.

CLI: `message` on stderr, exit code derived from the HTTP status class
(2 → 0, 4 → 1, 5 → 2). `--json` prints the envelope verbatim.

## 3. Registry

### Authentication and authorization

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_UNAUTHORIZED` | 401 | — | Missing or invalid bearer token |
| `ERR_FORBIDDEN` | 403 | — | Authenticated but not permitted for this action |
| `ERR_SCOPE_FORBIDDEN` | 403 | `scope` | Operation would read or write a scope the caller may not touch |

### Request shape

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_INVALID_REQUEST` | 400 | `detail` | Malformed body or unknown field |
| `ERR_FIELD_REQUIRED` | 400 | `field` | A required field is absent |
| `ERR_FIELD_INVALID` | 400 | `field`, `reason` | Present but violates a constraint |
| `ERR_UNSUPPORTED_LANG` | 400 | `lang` | Not a BCP-47 tag we accept |
| `ERR_PAYLOAD_TOO_LARGE` | 413 | `limit`, `actual` | Exceeds the documented limit |

### Entry and evidence

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_ENTRY_NOT_FOUND` | 404 | `uid` | No such entry or lineage |
| `ERR_EVIDENCE_REQUIRED` | 422 | `min` | An experience entry was submitted without evidence |
| `ERR_EVIDENCE_DANGLING` | 422 | `raw_uid` | The cited `raw` row does not exist |
| `ERR_EVIDENCE_IN_USE` | 409 | `raw_uid`, `owners` | Deleting a `raw` row that entries still cite |
| `ERR_ENTRY_CONFLICT` | 409 | `group`, `uids[]` | Same symptom, different fix; both are contested |
| `ERR_ENTRY_IMMUTABLE` | 409 | `uid`, `field` | Attempt to modify an append-only field of an experience |

Duplicate content is **not** an error. `mem_record` and `mem_remember` return the
existing `uid` with `status: "unchanged"` (or `"deduplicated"` when only the
evidence was merged).

### Concurrency and idempotency

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_WRITE_CONFLICT` | 409 | `uid`, `expected_rev`, `actual_rev` | Optimistic-lock failure: the row changed between read and write |
| `ERR_IDEMPOTENCY_MISMATCH` | 409 | `key` | The same `Idempotency-Key` was reused with a different request body |
| `ERR_PROJECT_UNRESOLVED` | 422 | `cwd` | No `project` row matches the working directory and no explicit scope was given |

`ERR_WRITE_CONFLICT` is only reachable in team mode. In solo mode there is a
single writer queue, so the conflict cannot occur; the code exists anyway so
that clients do not need two behaviours.

Replaying an `Idempotency-Key` with an **identical** request is not an error: the
cached response is returned with `200` and `idempotent_replay: true`. Only a
key/body mismatch is `ERR_IDEMPOTENCY_MISMATCH`.

`ERR_PROJECT_UNRESOLVED` is recoverable: the caller should retry with an explicit
`scope: "global"` or `scope: "session"`. Guessing a project from a path prefix is
forbidden, because in team mode two users' paths for the same repository differ.

### Policy

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_POLICY_SENSITIVITY` | 403 | `sensitivity`, `reason` | Content classified as never-persistable (L4) was rejected |
| `ERR_POLICY_REDACTION` | 422 | `rule_id` | Redaction could not complete; the write is refused rather than stored raw |

### Storage

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_DB_BUSY` | 503 | `retry_after_ms` | Writer lock contention; retry once, then give up |
| `ERR_DB_CORRUPT` | 500 | `detail` | `integrity_check` failed; restore from snapshot |
| `ERR_SCHEMA_VERSION` | 500 | `found`, `expected` | Store written by an incompatible version |
| `ERR_MIGRATION_FAILED` | 500 | `from`, `to`, `detail` | Migration rolled back; the pre-migration snapshot path is in `params` |
| `ERR_FTS_UNAVAILABLE` | 503 | `feature` | FTS5 missing or disabled; search is running in degraded `LIKE` mode |

### System

| Code | HTTP | Params | Meaning |
|---|---|---|---|
| `ERR_RATE_LIMITED` | 429 | `retry_after_ms` | Too many writes from one client |
| `ERR_NOT_IMPLEMENTED` | 501 | `feature` | Reserved surface, not yet available (e.g. `kinds: ["chunk"]` in P0) |
| `ERR_INTERNAL` | 500 | `request_id` | Unexpected failure; the id ties to the log line |

## 4. Agent guidance

Which codes an agent should react to, and how, is defined once in
`mcp-tools.md` §7. The short version: never retry a validation error
unchanged; never broaden a scope to work around a permission error; report
conflicts to the human rather than picking a side.

## 5. Change log

| Version | Change |
|---|---|
| 0.1.0 | Initial registry |
