#!/usr/bin/env node
// Keep the skill, its npm package, and the Go implementation in agreement.
//
// Checked:
//   1. skills/rocky-mem/SKILL.md version == npm/skill/package.json version
//   2. tools.json names and required parameters == internal/mcpserver/server.go
//   3. tools.json error codes ⊆ internal/errs/errs.go
//   4. tools.json contract is a positive integer
//
// Run in CI and before a release: `node scripts/check-skill-versions.mjs`.

import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const read = (rel) => readFile(join(root, rel), 'utf8');

const problems = [];

// 1. versions
const skill = await read('skills/rocky-mem/SKILL.md');
const frontmatter = skill.match(/^---\n([\s\S]*?)\n---/)?.[1] ?? '';
const skillVersion = frontmatter.match(/^version:\s*(\S+)$/m)?.[1];
const pkg = JSON.parse(await read('npm/skill/package.json'));
if (!skillVersion) problems.push('SKILL.md has no version in frontmatter');
else if (skillVersion !== pkg.version) {
  problems.push(`SKILL.md version ${skillVersion} != npm/skill/package.json ${pkg.version}`);
}

const tools = JSON.parse(await read('skills/rocky-mem/tools.json'));
if (!Number.isInteger(tools.contract) || tools.contract < 1) {
  problems.push('tools.json must declare a positive integer contract');
}

// 2. tool surface
const serverSrc = await read('internal/mcpserver/server.go');
const registered = [...serverSrc.matchAll(/Name:\s*"(mem_[a-z_]+)"/g)].map((m) => m[1]).sort();
const declared = (tools.tools ?? []).map((t) => t.name).sort();
if (registered.length === 0) problems.push('no mem_* tools found in internal/mcpserver/server.go');
if (JSON.stringify(registered) !== JSON.stringify(declared)) {
  problems.push(`tools.json tools ${JSON.stringify(declared)} != server ${JSON.stringify(registered)}`);
}

// Required parameters: the skill states them (they are part of the contract),
// so assert the server's arg structs actually carry those as non-omitempty tags.
const argStructFor = {
  mem_context: 'contextArgs',
  mem_search: 'searchArgs',
  mem_get: 'getArgs',
  mem_record: 'recordArgs',
  mem_remember: 'rememberArgs',
  mem_feedback: 'feedbackArgs',
};
for (const tool of tools.tools ?? []) {
  const struct = argStructFor[tool.name];
  if (!struct) continue;
  const block = serverSrc.match(new RegExp(`type ${struct} struct \\{([\\s\\S]*?)\\n\\}`))?.[1] ?? '';
  const requiredFields = [...block.matchAll(/json:"([a-z_]+)"/g)].map((m) => m[1]);
  for (const required of tool.required ?? []) {
    if (!requiredFields.includes(required)) {
      problems.push(`${tool.name}: required param "${required}" missing from ${struct}`);
    }
  }
}
const recordEvidence = (tools.tools ?? []).find((t) => t.name === 'mem_record')?.required ?? [];
if (!recordEvidence.includes('evidence')) problems.push('mem_record must require evidence');

// 3. error codes
const errsSrc = await read('internal/errs/errs.go');
const realCodes = new Set([...errsSrc.matchAll(/Code = "([A-Z_]+)"/g)].map((m) => m[1]));
for (const code of tools.error_codes ?? []) {
  if (!realCodes.has(code)) problems.push(`tools.json lists unknown error code ${code}`);
}

if (problems.length > 0) {
  for (const p of problems) console.error(`skill check: ${p}`);
  process.exit(1);
}
console.log(
  `skill check ok: version ${skillVersion}, contract ${tools.contract}, ` +
    `${tools.tools.length} tools, ${tools.error_codes.length} error codes`,
);
