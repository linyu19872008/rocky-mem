# rocky-mem-skill

The portable skill for [rocky-mem](https://github.com/linyu19872008/rocky-mem),
published from the server repository so the skill and the contract it documents
share one version.

```sh
# Cursor (project rule)
npx rocky-mem-skill install --host cursor

# Claude Code (user skill)
npx rocky-mem-skill install --host claude-code

# Codex / any AGENTS.md host (project, idempotent)
npx rocky-mem-skill install --host codex

# DeepSeek Harness (user skill; the bundle does this too)
npx rocky-mem-skill install --host dsh
```

`--cwd PATH` targets another project, `--dry-run` prints what would be written.

The bundle `rocky-mem-dsh` depends on this package and registers the skill
directly, so DSH users normally do not need to run the installer.

Source of truth: `skills/rocky-mem/` in the rocky-mem repository. This package
is regenerated from it at publish time (`prepack`), never edited by hand.
