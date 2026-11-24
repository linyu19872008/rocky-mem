# Agent instructions — rocky-mem

This repository provides a local memory/experience service over MCP.
Treat rocky-mem as the project’s **experience layer**. Cursor Rules / this file
hold standing instructions; rocky-mem holds searchable lessons with provenance.

## Required MCP tools

| Tool | When |
|---|---|
| `mem_context` | Start of every non-trivial session (cheap; call unconditionally) |
| `mem_search` | Before starting a non-trivial task |
| `mem_get` | When a pack item needs full evidence or neighbours |
| `mem_record` | After you learn a reusable engineering lesson (evidence mandatory) |
| `mem_remember` | When a standing preference/fact/constraint changes |
| `mem_feedback` | When a retrieved entry was helpful, wrong, or stale |

## Protocol (do not skip)

1. **Session start:** call `mem_context` with the working directory `cwd`.
2. **Before non-trivial work:** call `mem_search` with a concrete query (error
   text, symptom, or goal). Prefer searching over guessing from training data.
3. **Respect the pack:** honour `## Preferences`. For `## Applies`, check
   `Applies when` / `Do not apply when`. Treat `## Do not use` as hard negatives.
4. **If reality conflicts with an entry:** call `mem_feedback` with
   `outcome=wrong` and a short `note` — do not silently ignore it.
5. **When you finish a lesson worth reusing:** call `mem_record` with at least
   one evidence item (`quote` / `commit` / `path`). Include `applies_when` and
   `invalid_when` whenever you can. Do not claim success if status is
   `pending_review` or `rejected`.
6. **Standing rules for this repo:** call `mem_remember` (use a stable
   `mem_key` when updating).

## What belongs where

- **Remember (memory):** current preferences and constraints — “use pnpm”.
- **Record (experience):** conditional lessons — “ConfigMap file mounts lag
  during short rollouts; checksum annotation; not for API-read configs”.
- **Do not store:** secrets, tokens, private keys (redaction will reject L4).

## Trivial tasks (skip search)

One-line renames, pure formatting, or questions answered entirely by open files
may skip `mem_search`. Prefer still calling `mem_context` once per session.
