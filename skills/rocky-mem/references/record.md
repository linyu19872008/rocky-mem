# Writing an experience

`mem_record` is a claim, not a fact. Write it so a stranger could verify it.

## Fields

| Field | What belongs there | Common mistake |
|---|---|---|
| `title` | the lesson in one line, symptom-shaped | "K8s bug" |
| `exp_type` | `howto` · `pitfall` · `decision` · `postmortem` | using `postmortem` for a two-line fix |
| `symptom` | observable evidence: error text, wrong output | the whole log |
| `root_cause` | why it happens; prefix `hypothesis:` when unsure | asserting a cause you did not verify |
| `fix` | exact command or config, Markdown | "restart the thing" |
| `applies_when` | the conditions that make this true | "general", "always" |
| `invalid_when` | where this is wrong or harmful | leaving it empty |
| `outcome` | `success` · `failed` · `unknown` | marking unverified work `success` |
| `confidence` | 0–1, earned by verification | 0.9 by habit |
| `evidence` | at least one item: `quote` / `commit` / `path` / `url` / `session` | inventing a commit sha |

`invalid_when` is the most valuable field in the table. Agents rarely fail
because they do not know a fix; they fail because they applied it where it does
not hold.

## Auto-publish gate

An entry publishes immediately only when **all** of these hold:

1. evidence resolves to at least one raw record;
2. `applies_when` is non-empty;
3. `confidence >= 0.6`;
4. no `symptom_key` conflict with an existing entry;
5. `sensitivity <= 2`.

Otherwise it is created with `trust=proposed` and enters the review queue. The
response says so; repeat that status to the user.

## Anti-patterns that get entries rejected or ignored

- Pasting a stack trace into `symptom` (use the signature line).
- `applies_when: general` — indistinguishable from "I do not know".
- Evidence that points at a file you did not open.
- Recording a lesson twice with different wording: content hashing catches the
  exact duplicate, not the paraphrase. Search first.
- Recording before the fix is verified. `outcome: unknown` is honest and
  welcome; a false `success` is what poisons the ledger.

## Memories are different

`mem_remember` is for **state**, not lessons: "we use pnpm here", "the staging
API rejects tokens older than 10 minutes". Provide `mem_key` to get
update-in-place semantics; the previous value is superseded, never deleted.
Supply `rev` only when you want compare-and-swap and a loud
`ERR_WRITE_CONFLICT` instead of last-write-wins.
