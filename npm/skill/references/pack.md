# Reading a context pack

`mem_search` returns Markdown with four fixed English headings. The headings
are stable on purpose: prompt caching works, output is comparable across
agents and languages, and evaluation can assert on structure. The body stays in
whatever language the entry was written in.

## `## Preferences`

Standing memory for the resolved scope: preferences, constraints, facts, terms.
Each line carries a scope and a status (`pinned`, `active`, uid).

Read as **hard constraints**. If one contradicts reality, call `mem_feedback`
with `outcome=wrong` and a note — do not quietly work around it.

## `## Applies`

Experience entries whose conditions plausibly match. Each block:

```
### [pitfall] Title
- Applies when: the conditions under which this is true
- Do: the concrete action
- Do not apply when: the boundary where it is wrong or harmful
- Evidence: commit abc1234 (L2 · active · confidence 0.85 · e_01J9K…)
```

Use it like this:

1. Match your situation against `Applies when`. If it does not match, skip the
   entry — do not stretch it.
2. Check `Do not apply when` **before** acting. A boundary match means stop.
3. Trust `confidence` and `Evidence`: an entry at 0.5 with a single commit is a
   hypothesis; at 0.85 with a reproduction it is a working recipe.
4. `contested` entries appear in pairs. Report both to the human with their
   uids instead of picking one.

## `## Do not use`

Populated from entries with `outcome=failed` and from disputed pairs. Treat as
hard negatives: these were tried and did not work, or are currently in dispute.
Repeating them is the most expensive failure mode this system exists to prevent.

## `## Background`

Corpus excerpts for orientation. Never a plan, never a claim. If you need the
whole document, ask for it explicitly rather than inferring from the excerpt.

## Truncation

When the pack hits its token budget it ends with a line stating how many items
were omitted and that `mem_get` retrieves individual entries. If the omitted
count is large, narrow the query instead of raising the budget blindly.
