# Troubleshooting rocky-mem

## `pending_review` came back

Not a failure. The gate wants evidence, `applies_when`, or confidence ≥ 0.6.
Read `reason` in the response, fix that one thing, and record again. Do not
retry unchanged, and do not tell the user it was saved.

## Two entries are `contested`

The server found the same symptom with a different fix. Both are returned and
both are demoted. Report the pair with uids to the human; the resolution is a
human decision (or a `mem_feedback` from whoever reproduces it).

## `degraded:*` in `notice`

| Notice | Meaning | What to do |
|---|---|---|
| `degraded:fts_unavailable` | full-text index missing, `LIKE` fallback | results may be incomplete; say so |
| `degraded:project_unresolved` | git remote not resolvable | pass `scope: global` or `session` |
| `degraded:vector_unavailable` | semantic path off | lexical results only |

## `ERR_PROJECT_UNRESOLVED`

`cwd` was resolved, but the git remote was not. Do not guess a project from a
path prefix — that is how cross-project leakage happens. Retry with
`scope: "global"` or `scope: "session"`.

## `ERR_FTS_UNAVAILABLE`

Search works but is degraded. Report the degradation instead of presenting a
short result list as complete.

## `ERR_EVIDENCE_REQUIRED`

`mem_record` needs at least one evidence item that resolves. Add a quote of what
you actually saw, or a commit/path you actually touched.

## `ERR_WRITE_CONFLICT`

You supplied `rev` and the value changed. Re-read with `mem_get`, re-apply on
the current `rev`. Never blind-retry.

## The service is not running

Symptoms: injection blocks do not appear, `mem_*` tools fail to connect. Ask the
user to start it (`rockymem serve`). Do not try to work around it by writing
lessons into a file that looks like memory storage.
