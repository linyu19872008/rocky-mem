# Claude Code

Copy this skill into your user skills directory:

```sh
mkdir -p ~/.claude/skills/rocky-mem
cp ../../SKILL.md ~/.claude/skills/rocky-mem/SKILL.md
cp -R ../../references ~/.claude/skills/rocky-mem/
```

Or run the installer:

```sh
npx rocky-mem-skill install --host claude-code
```

Then connect the MCP server (stdio):

```sh
claude mcp add rocky-mem -- rockymem mcp
```
