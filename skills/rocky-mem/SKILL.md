---
name: rocky-mem
description: Consult and maintain rocky-mem, the engineering experience ledger — standing preferences, provenance-backed lessons, and the conditions where those lessons stop applying. Use before non-trivial debugging, design, or ops work, and after learning a reusable lesson.
version: 0.1.0
requires-contract: 1
---

# rocky-mem

A ledger, not a log. It stores two different things:

- **memory** — state. Preferences, facts, constraints, glossary. One current
  value per key, with a version chain.
- **experience** — a ledger of claims. "This symptom, this root cause, this fix,
  valid under these conditions, invalid under those." Never overwritten;
  contradictions stay visible as `contested` pairs.

Your job is to consult it before guessing, and to write into it only when you
learnt something reusable.

## When to use it

| Situation | Call |
|---|---|
| A new non-trivial session starts | `mem_context` with the working directory |
| Before debugging, designing, or changing ops config | `mem_search` with the symptom or error text |
| A pack item needs its evidence or neighbours | `mem_get` |
| You finished a task and learnt something reusable | `mem_record` |
| A standing preference/fact/constraint changed | `mem_remember` |
| A retrieved item helped, misled you, or is stale | `mem_feedback` |

## When NOT to use it

- Rename-a-variable, formatting, or single-line typo work — no lookup needed.
- The user explicitly said not to consult or write memory.
- You are about to record something you did not observe (a guess, a general
  best practice, something you read in training data). `mem_record` requires
  evidence; a guess without evidence should stay in the conversation.
- You want to store credentials, tokens, customer data, or anything the user
  asked you not to remember.

## The protocol

### 1. Session start — `mem_context`

Call it once with `cwd`. Read `## Preferences` as **hard constraints**: they
are current state, not suggestions. If a preference contradicts what you
observe, that is a signal to call `mem_feedback`, not to ignore it.

### 2. Before work — `mem_search`

Query with what you can actually observe: the error text, the unexpected
behaviour, the component name. Not with your hypothesis — a hypothesis-shaped
query retrieves entries that agree with the hypothesis.

Read the pack in this order:

- `## Preferences` — constraints.
- `## Applies` — candidates. **Check `Applies when` and `Do not apply when`
  before acting.** This is the whole point of the ledger.
- `## Do not use` — hard negatives. These are approaches that were tried and
  failed, or entries currently disputed. Do not repeat them.
- `## Background` — context only, never a plan.

See `references/pack.md` for the full reading rules.

### 3. After work — `mem_record`

Record only when the lesson is reusable by someone else in this project. One
entry per lesson. Evidence is mandatory.

Auto-publish requires **all** of: evidence that resolves, non-empty
`applies_when`, `confidence >= 0.6`, no conflict with an existing entry, and
`sensitivity <= 2`. Otherwise you get `pending_review`.

`pending_review` is **not** success. Say so plainly: "recorded as pending
review" — never "saved" or "remembered".

See `references/record.md` for field-by-field guidance and the anti-patterns
that get entries rejected.

### 4. When reality disagrees — `mem_feedback`

| Outcome | Effect |
|---|---|
| `helpful` | use count += 1, last-used timestamp |
| `wrong` | trust `active → contested`, note stored, ranking demoted immediately |
| `stale` | flag; three independent reports in 90 days moves it to `stale` |

`outcome=wrong` requires a `note` saying what actually happened. One `wrong`
never deletes anything — an agent's "wrong" is often a misapplication, which is
exactly the signal that `invalid_when` needs writing.

## Rules that matter

1. **Never claim success on `pending_review` or `rejected`.**
2. **Never silently discard a contradiction.** Either the entry is wrong, or
   your situation is outside its `applies_when` — both are worth a feedback.
3. **Do not paste logs wholesale.** `symptom` is the signature, not the transcript.
4. **Do not set `confidence: 0.9` by habit.** 0.9 means "I reproduced this and
   verified the fix"; most lessons are 0.6–0.8.
5. **Do not write `applies_when: general`.** "General" is the thing you are
   trying to avoid claiming.
6. **Respect scope.** `project` entries come from the git remote, not from a
   path prefix. If the project cannot be resolved you get
   `ERR_PROJECT_UNRESOLVED` — pass `scope: global` or `session`, never widen
   silently.

## Tool names

Hosts may prefix tool names (`mcp__rockymem__mem_search` in some, plain
`mem_search` in others). Match on the `mem_` suffix, never on the prefix.

## Troubleshooting

`references/troubleshoot.md` covers `pending_review`, `contested`, `degraded:*`,
`ERR_FTS_UNAVAILABLE`, and "the service is not running".
