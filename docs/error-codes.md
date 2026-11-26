# Error codes

A stable public contract. Clients branch on `code`; humans read `message`.

**This registry mirrors `internal/errs/errs.go` exactly.** If a code is not a
constant there, it does not exist — no matter what an older draft of this
document said.

## 1. Conventions

- Format: `ERR_<DOMAIN>_<CONDITION>`, `SCREAMING_SNAKE_CASE`, ASCII.
- Codes are **never renamed, never reused, never repurposed.** A code that stops
  being emitted stays in this table, marked *reserved*, for at least one major
  version.
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
    "message": "At least one evidence item is required",
    "params": { "min": 1 }
  }
}
```

MCP: the same object, JSON-encoded, inside a text content block, with
`isError: true`.

CLI: `message` on stderr, exit code derived from the HTTP status class
(2 → 0, 4 → 1, 5 → 2). `--json` prints the envelope verbatim.

## 3. Registry

*Emitted* means the code is actually returned by shipped code in solo mode.
*Reserved* means the constant exists but nothing returns it today; clients must
still tolerate it (it may appear in a future release without a breaking change).

| Code | HTTP | Status | Meaning |
|---|---|---|---|
| `ERR_UNAUTHORIZED` | 401 | emitted | Missing or invalid bearer token / session |
| `ERR_FORBIDDEN` | 403 | **reserved** | Authenticated but not permitted. Team-mode concept; no caller can be forbidden in solo |
| `ERR_SCOPE_FORBIDDEN` | 403 | **reserved** | Operation would touch a scope the caller may not access. Team-mode only |
| `ERR_INVALID_REQUEST` | 400 | emitted | Malformed body, unknown field, unparsable JSON |
| `ERR_FIELD_REQUIRED` | 400 | emitted | A required field is absent (`params.field`) |
| `ERR_FIELD_INVALID` | 400 | emitted | Present but violates a constraint (`params.field`, `params.reason`) |
| `ERR_UNSUPPORTED_LANG` | 500 | **reserved** | Not a BCP-47 tag we accept |
| `ERR_PAYLOAD_TOO_LARGE` | 500 | **reserved** | Exceeds a documented limit |
| `ERR_EVIDENCE_REQUIRED` | 400 | emitted | An experience entry was submitted without evidence |
| `ERR_WRITE_CONFLICT` | 409 | emitted | `rev` CAS failed: the row changed between read and write (`params.uid`, `params.expected_rev`, `params.actual_rev`) |
| `ERR_NOT_FOUND` | 404 | emitted | No such entry, lineage, or raw record (`params.uid`) |
| `ERR_PROJECT_UNRESOLVED` | 500 | emitted | No `project` row matches `cwd` and no explicit scope was given |
| `ERR_POLICY_REJECTED` | 400 | emitted | Content refused by the redaction policy (L4, unredactable) (`params.reject_why`) |
| `ERR_FTS_UNAVAILABLE` | 500 | **reserved** | FTS5 missing; search runs in degraded `LIKE` mode |
| `ERR_NOT_IMPLEMENTED` | 501 | emitted | Reserved surface, e.g. `kinds: ["chunk"]` |
| `ERR_DB` | 500 | **reserved** | Storage-layer failure |
| `ERR_SYS_INTERNAL` | 500 | emitted | Unexpected failure; also the wrapper for any unclassified error |

### Notes that matter to a client

1. **`ERR_WRITE_CONFLICT` is reachable in solo mode.** Two clients that each read
   the same `rev` and then write back will hit it. It is not a team-mode-only
   code; treat it as "re-read, then retry".
2. **Status codes are coarse.** Four codes whose meaning is client-error-ish
   (`ERR_UNSUPPORTED_LANG`, `ERR_PAYLOAD_TOO_LARGE`, `ERR_FTS_UNAVAILABLE`,
   `ERR_PROJECT_UNRESOLVED`) currently fall through to `500` because the status
   mapping only special-cases a subset. Branch on `code`, never on the status.
   Tightening this is tracked as a wart, not a contract change.
3. **Deep links to other documents:** `docs/mcp-tools.md` §7 says how an agent
   should react to each of these.

### Deliberately absent

Codes from earlier drafts that **do not exist** and must not be reintroduced
without adding the constant first:

`ERR_ENTRY_NOT_FOUND` (use `ERR_NOT_FOUND`), `ERR_ENTRY_CONFLICT`,
`ERR_EVIDENCE_DANGLING`, `ERR_EVIDENCE_IN_USE`, `ERR_ENTRY_IMMUTABLE`,
`ERR_IDEMPOTENCY_MISMATCH`, `ERR_RATE_LIMITED`, `ERR_POLICY_SENSITIVITY`,
`ERR_POLICY_REDACTION`, `ERR_DB_BUSY`, `ERR_DB_CORRUPT`, `ERR_SCHEMA_VERSION`,
`ERR_MIGRATION_FAILED`, `ERR_INTERNAL`.

Two of them existed only because a feature was planned: `ERR_IDEMPOTENCY_MISMATCH`
(the `idempotency` table is built but not wired) and `ERR_RATE_LIMITED` (no rate
limiter). Others were speculative.

## 4. Agent guidance

| Code | Agent behaviour |
|---|---|
| `ERR_EVIDENCE_REQUIRED` | Add evidence; do not retry unchanged |
| `ERR_FIELD_REQUIRED` / `ERR_FIELD_INVALID` | Fix the named field; do not retry unchanged |
| `ERR_WRITE_CONFLICT` | Re-read with `mem_get`, re-apply on the current `rev`; never blind-retry |
| `ERR_PROJECT_UNRESOLVED` | Retry with `scope: "global"` or `scope: "session"`; do not guess a project from the path |
| `ERR_NOT_IMPLEMENTED` | The surface is reserved; report it, do not work around it |
| `ERR_POLICY_REJECTED` | Do not retry. The content was refused by policy; tell the human |
| `ERR_UNAUTHORIZED` | Stop. The client needs a token, not a retry |
| `ERR_NOT_FOUND` | Re-resolve the `uid`; it may have been purged |

## 5. Change log

| Version | Change |
|---|---|
| 0.1.2 | Registry realigned to `internal/errs/errs.go`; 14 phantom codes removed; `ERR_WRITE_CONFLICT` corrected to "reachable in solo" |
| 0.1.0 | Initial registry |
