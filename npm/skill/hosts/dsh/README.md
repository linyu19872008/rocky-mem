# DeepSeek Harness

Install the bundle instead of copying files — it registers this skill, injects
`mem_context` at session start, and wires the six MCP tools:

```sh
dsh plugin --profile web add rocky-mem-dsh
```

The bundle reads this directory's `SKILL.md` from the `rocky-mem-skill` npm
package (an optional dependency). Without that package it falls back to a short
embedded body, so the skill is always at least minimally present.

See <https://github.com/linyu19872008/rocky-mem-dsh>.
