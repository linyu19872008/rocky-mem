# MCP tool contracts

Agent-facing surface. Six tools, deliberately few: every additional tool is
another thing an agent can misuse.

**All identifiers, parameter names and descriptions are English and are never
localized.** Locale affects human-facing surfaces only (`ADR-0004`).

Transport: `rockymem mcp --stdio` (thin proxy to the local daemon) or
`POST /mcp` for HTTP clients.

---

## 0. Cross-cutting rules

| Topic | Rule |
|---|---|
| Determinism | Identical inputs against an unchanged store return identical output, byte for byte |
| Errors | Return `isError: true` with a structured payload `{code, message, params}`; `code` is stable and listed in `error-codes.md` |
| Side effects | `mem_search`, `mem_context`, `mem_get` have no visible side effects. They write an `audit` row and may bump `use_count`; they never write knowledge |
| Idempotency | `mem_remember` and `mem_record` are idempotent on content hash |
| Latency budget | read tools p95 ≤ 30 ms; write tools p95 ≤ 20 ms |
| Payload budget | a single tool result must not exceed 8 000 tokens; `mem_search` self-limits via `budget_tokens` |
| Truncation | always announced in the payload; never silent |
| Concurrency | writes carry `rev`. In team mode a stale `rev` returns `ERR_WRITE_CONFLICT`; in solo mode the write always succeeds because there is one writer |
| Project resolution | `cwd` is resolved to a `project` through the git remote, never by path prefix. If it cannot be resolved, the write fails with `ERR_PROJECT_UNRESOLVED` rather than silently widening scope |

### Tool description protocol

The following sentences are part of the tool descriptions, verbatim. Agents read
tool descriptions, not documentation.

> **mem_search** — "Search the local knowledge base before starting a non-trivial
> task. Returns a budgeted context pack with provenance. If an entry conflicts
> with what you actually observe, call mem_feedback with outcome=wrong instead of
> silently ignoring it."

> **mem_record** — "Record a reusable engineering lesson. Evidence is mandatory:
> cite what you actually observed. Entries are reviewed before they become
> authoritative; the response tells you the real status — do not claim success if
> status is pending_review or rejected."

These two sentences encode the behavioural contract. They are not marketing copy.

---

## 1. `mem_context`

Session-start injection. Returns the standing memory for a working directory plus
any pinned experience, under a small fixed budget. Cheap enough to call
unconditionally on every session.

**Input**

```json
{
  "type": "object",
  "properties": {
    "cwd":   { "type": "string", "description": "Absolute path of the working directory; used to infer project scope." },
    "task_hint": { "type": "string", "maxLength": 300, "description": "Optional one-line description of the task, used only to prioritise pinned entries." },
    "budget_tokens": { "type": "integer", "minimum": 200, "maximum": 1200, "default": 600 }
  },
  "additionalProperties": false
}
```

All fields optional. Calling with no arguments returns global-scope memory only.

**Output**

```json
{
  "pack": "## Preferences\n- pnpm; never npm (project · pinned · m_01J8Z…)\n…",
  "uids": ["m_01J8Z…", "m_01J8Y…"],
  "scope_resolved": { "scope": "project", "project_id": 42, "project": "corp/service-a" },
  "truncated": false,
  "notice": null
}
```

`notice` carries a machine-readable string when the store is degraded, e.g.
`degraded:fts_unavailable`. Agents may ignore it; the UI must not.

---

## 2. `mem_search`

The workhorse. Returns a sectioned, budgeted Markdown pack.

**Input**

```json
{
  "type": "object",
  "properties": {
    "query":  { "type": "string", "minLength": 1, "maxLength": 500 },
    "cwd":    { "type": "string" },
    "scope":  { "type": "string", "enum": ["global", "team", "project", "session"] },
    "kinds":  { "type": "array", "items": { "type": "string", "enum": ["memory", "experience", "chunk"] },
                "default": ["memory", "experience"] },
    "lang":   { "type": "string", "description": "BCP-47 filter for content language. Omit to search all languages." },
    "include_proposed": { "type": "boolean", "default": true,
                          "description": "Include entries awaiting review, at reduced rank." },
    "budget_tokens": { "type": "integer", "minimum": 200, "maximum": 4000, "default": 1200 },
    "limit":  { "type": "integer", "minimum": 1, "maximum": 50, "default": 12 }
  },
  "required": ["query"],
  "additionalProperties": false
}
```

**Output**

```json
{
  "pack": "## Preferences\n…\n\n## Applies\n### [pitfall] …\n- Applies when: …\n- Do: …\n- Do not apply when: …\n- Evidence: commit abc1234 (L2 · active · confidence 0.85 · e_01J9K…)\n\n## Do not use\n- …\n\n## Background\n- …",
  "sections": [
    { "name": "preferences", "count": 2 },
    { "name": "applies",     "count": 3 },
    { "name": "do_not_use",  "count": 1 },
    { "name": "background",  "count": 0 }
  ],
  "uids": ["e_01J9K…", "e_01J9J…", "m_01J8Z…"],
  "truncated": true,
  "omitted": 7,
  "expansion": { "terms": ["显存", "VRAM"], "source": "term_map" },
  "request_id": "req_01J9M…"
}
```

**Pack format.** Headings are English and stable:
`## Preferences`, `## Applies`, `## Do not use`, `## Background`.
Each experience item lists `Applies when`, `Do`, `Do not apply when`, `Evidence`.
`Do not use` is populated from `outcome='failed'` entries and from `contested`
pairs. When truncated, the pack ends with a line stating how many items were
omitted and that `mem_get` retrieves individual entries.

`expansion` is returned only when term expansion actually fired, so an agent can
tell that the results were reached indirectly.

---

## 3. `mem_get`

Full record for a single entry, including its evidence chain and neighbours.

**Input**

```json
{
  "type": "object",
  "properties": {
    "uid": { "type": "string", "description": "m_… / e_… / a lineage uid. For memory, always resolves to the current active version." },
    "include_raw": { "type": "boolean", "default": false,
                     "description": "Include truncated excerpts of the underlying evidence." }
  },
  "required": ["uid"],
  "additionalProperties": false
}
```

**Output**

```json
{
  "type": "experience",
  "uid": "e_01J9K…",
  "entry": { "exp_type": "pitfall", "title": "…", "symptom": "…", "root_cause": "…",
             "fix_md": "…", "applies_when": "…", "invalid_when": "…",
             "outcome": "success", "confidence": 0.85, "trust": "active",
             "scope": "project", "scope_ref": "/Users/x/repo/service-a",
             "lang": "en", "sensitivity": 2,
             "created_by": "dsh", "t_created": 1795440496, "use_count": 4 },
  "evidence": [ { "raw_uid": "r_01J9A…", "channel": "git", "quote": "…",
                  "role": "support", "sensitivity": 2 } ],
  "tags": ["k8s", "configmap"],
  "related": [ { "uid": "e_01J9J…", "relation": "contradicts", "title": "…" } ],
  "lineage": null
}
```

For memory the `entry` object contains `mem_key`, `mem_type`, `body_md`,
`status`, `pinned`, `priority` and `lineage`, plus `versions` — a list of prior
versions by `uid` and `created_at`.

---

## 4. `mem_remember`

Write or update a memory. Upsert semantics: one current value per
`(scope, scope_ref, mem_key)`.

**Input**

```json
{
  "type": "object",
  "properties": {
    "title":   { "type": "string", "minLength": 1, "maxLength": 200 },
    "body":    { "type": "string", "minLength": 1, "maxLength": 4000,
                 "description": "Markdown. State the rule, not the reasoning." },
    "mem_key": { "type": "string", "maxLength": 80, "pattern": "^[a-z0-9_.-]+$",
                 "description": "Optional stable key such as 'package_manager'. Providing a key enables update-in-place semantics." },
    "mem_type": { "type": "string", "enum": ["pref", "fact", "constraint", "glossary"], "default": "fact" },
    "scope":   { "type": "string", "enum": ["global", "team", "project", "session"] },
    "scope_ref": { "type": "string" },
    "pinned":  { "type": "boolean", "default": false },
    "priority": { "type": "integer", "minimum": -10, "maximum": 10, "default": 0 },
    "rev":     { "type": "integer",
                 "description": "Optimistic-lock token. Omit for last-write-wins; supply it (from mem_get) for compare-and-swap." },
    "tags":    { "type": "array", "items": { "type": "string", "maxLength": 40 }, "maxItems": 10 },
    "source_quote": { "type": "string", "maxLength": 2000,
                      "description": "Optional origin: where this was stated." }
  },
  "required": ["title", "body"],
  "additionalProperties": false
}
```

**Output**

```json
{
  "uid": "m_01J8Z…",
  "lineage": "m_01J8Z…",
  "status": "created",
  "rev": 1,
  "superseded_uid": null,
  "review_required": false
}
```

`status` is `created`, `updated` (an existing current version was superseded),
`unchanged` (content hash identical), or `pending_review`.

Memory publishes immediately by default. It enters `pending_review` when
`sensitivity = 3`, or when the scalar value contradicts a `pinned` entry in the
same scope.

**Two write modes, chosen by whether `rev` is supplied:**

| `rev` | Semantics | On conflict |
|---|---|---|
| omitted | Last-write-wins. The current value is superseded and the new row becomes active | Never conflicts; both versions stay in the lineage |
| supplied | Compare-and-swap against that exact version | `ERR_WRITE_CONFLICT` with `expected_rev` and `actual_rev` |

Omitting `rev` is the normal case for a preference ("the package manager is now
pnpm"). Supplying it is for corrections where a concurrent edit would be lost
silently — the kind of change a human would want to re-read before overwriting.
In solo mode both paths behave identically, because there is one writer.

---

## 5. `mem_record`

Write an experience. Always treated as a claim, never as fact.

**Input**

```json
{
  "type": "object",
  "properties": {
    "title":  { "type": "string", "minLength": 1, "maxLength": 200 },
    "exp_type": { "type": "string", "enum": ["howto", "pitfall", "decision", "postmortem"] },
    "symptom": { "type": "string", "maxLength": 2000,
                 "description": "What is observable: the error text, the wrong behaviour. Prefer verbatim output." },
    "root_cause": { "type": "string", "maxLength": 2000,
                    "description": "Why it happens. If uncertain, prefix with 'hypothesis:'." },
    "fix": { "type": "string", "maxLength": 6000,
             "description": "The concrete action, including the exact command or config. Markdown." },
    "applies_when": { "type": "string", "maxLength": 1000 },
    "invalid_when": { "type": "string", "maxLength": 1000,
                      "description": "Conditions where this advice is wrong or harmful. Strongly recommended." },
    "outcome": { "type": "string", "enum": ["success", "failed", "unknown"], "default": "unknown" },
    "confidence": { "type": "number", "minimum": 0, "maximum": 1, "default": 0.5 },
    "scope": { "type": "string", "enum": ["global", "team", "project", "session"] },
    "scope_ref": { "type": "string" },
    "evidence": {
      "type": "array", "minItems": 1, "maxItems": 10,
      "items": {
        "type": "object",
        "properties": {
          "type": { "type": "string", "enum": ["quote", "commit", "path", "url", "session"] },
          "ref":  { "type": "string", "description": "Commit sha, path, URL or session id. Omitted for type=quote." },
          "text": { "type": "string", "maxLength": 2000, "description": "Verbatim excerpt. Required for type=quote." }
        },
        "required": ["type"]
      }
    },
    "tags": { "type": "array", "items": { "type": "string", "maxLength": 40 }, "maxItems": 10 }
  },
  "required": ["title", "exp_type", "evidence"],
  "additionalProperties": false
}
```

**Evidence resolution**

Best-effort and shallow (no deep git object walk, no remote fetch):

| `type` | Server behaviour |
|---|---|
| `quote` | Creates a `raw` row with `channel='agent'`, storing the verbatim text. Redaction applies. |
| `commit` | If `cwd` is a git work tree, runs `git log -1` for `ref` and stores sha / subject / author / date. Otherwise stores `text` or `commit <ref>` and adds a `warnings` entry. |
| `path` | If `ref` exists under `cwd`, stores repo-relative path + size + mtime. Otherwise stores `text` or `path <ref>` with a warning. |
| `url` | Stored as a raw reference; the server does not fetch it. |
| `session` | Stores `text` or `session <ref>`; no implicit session injection in solo mode. |

If resolution fails, the evidence is still persisted as text when possible and the
response includes a `warnings` entry. Empty evidence after resolution is skipped
(and may trigger `ERR_EVIDENCE_REQUIRED` if nothing remains).

**Output**

```json
{
  "uid": "e_01J9K…",
  "rev": 1,
  "status": "pending_review",
  "trust": "proposed",
  "review_required": true,
  "conflict": null,
  "warnings": [],
  "reason": "applies_when missing"
}
```

`status` is `published`, `pending_review`, or `rejected`.

Auto-publish requires **all** of:

1. `evidence` resolves to at least one `raw` row
2. `applies_when` is non-empty
3. `confidence >= 0.6`
4. no `symptom_key` conflict with an existing entry
5. `sensitivity <= 2`

Otherwise the entry is created with `trust='proposed'` and appears in the review
queue. `rejected` is returned only when content is L4, the payload exceeds
limits, or redaction cannot complete.

When a conflict is detected, `conflict` is populated and **both** entries move to
`contested`:

```json
"conflict": { "group": "cg_01J9N…", "uids": ["e_01J9K…", "e_01J8Q…"],
              "reason": "same symptom_key, different fix_hash" }
```

---

## 6. `mem_feedback`

The fuel of the falsification mechanism. Cheap; agents should call it freely.

**Input**

```json
{
  "type": "object",
  "properties": {
    "uid": { "type": "string" },
    "outcome": { "type": "string", "enum": ["helpful", "wrong", "stale"] },
    "note": { "type": "string", "maxLength": 1000,
              "description": "Required when outcome=wrong: what actually happened." }
  },
  "required": ["uid", "outcome"],
  "additionalProperties": false
}
```

**Effect**

| outcome | Effect |
|---|---|
| `helpful` | `use_count += 1`, `t_last_used = now` |
| `wrong` | `trust: active → contested`; a `feedback` row with the note; the entry is demoted in ranking immediately |
| `stale` | flag raised; three independent `stale` reports within 90 days moves the entry to `trust='stale'` |

A single `wrong` does not delete anything. Deletion requires human action, because
an agent's "wrong" is frequently a misapplication of a correct entry — which is
itself the signal that `invalid_when` needs writing.

---

## 7. Error mapping

MCP errors return `isError: true` and a text content block containing JSON:

```json
{ "code": "ERR_EVIDENCE_REQUIRED", "message": "At least 1 evidence item is required for an experience entry.", "params": { "min": 1 } }
```

`message` is English and stable enough for an agent to reason about. Full
registry: `error-codes.md`.

Codes an agent should handle explicitly:

| Code | Agent behaviour |
|---|---|
| `ERR_EVIDENCE_REQUIRED` | Add evidence; do not retry unchanged |
| `ERR_ENTRY_CONFLICT` | Stop; report the conflict to the human with both uids |
| `ERR_SCOPE_FORBIDDEN` | Do not retry with a broader scope |
| `ERR_WRITE_CONFLICT` | Re-read with `mem_get`, re-apply on the current `rev`; never blind-retry |
| `ERR_PROJECT_UNRESOLVED` | Retry with `scope: "global"` or `scope: "session"`; do not guess a project from the path |
| `ERR_FTS_UNAVAILABLE` | Search is degraded; results may be incomplete. Say so. |
| `ERR_DB_BUSY` | Retry once after a short delay, then give up |
