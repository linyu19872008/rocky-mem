#!/usr/bin/env node
// rocky-mem-skill installer.
//
//   npx rocky-mem-skill install --host cursor|claude-code|codex|dsh
//
// Each host gets the format it actually reads:
//   cursor      .cursor/rules/rocky-mem.mdc          (project)
//   claude-code ~/.claude/skills/rocky-mem/          (user)
//   codex       AGENTS.md marked block               (project, idempotent)
//   dsh         ~/.dsh/skills/rocky-mem/             (user, filesystem provider)

import { mkdir, readFile, writeFile, cp, access } from 'node:fs/promises';
import { homedir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const argv = process.argv.slice(2);

function flag(name, fallback) {
  const i = argv.indexOf(`--${name}`);
  return i >= 0 && argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[i + 1] : fallback;
}

const command = argv[0] ?? 'install';
const host = flag('host', '');
const cwd = resolve(flag('cwd', process.cwd()));
const dryRun = argv.includes('--dry-run');

const HOSTS = ['cursor', 'claude-code', 'codex', 'dsh'];

async function exists(p) {
  try {
    await access(p);
    return true;
  } catch {
    return false;
  }
}

async function put(target, content) {
  if (dryRun) {
    console.log(`[dry-run] would write ${target}`);
    return;
  }
  await mkdir(dirname(target), { recursive: true });
  await writeFile(target, content, 'utf8');
  console.log(`wrote ${target}`);
}

async function copy(from, to) {
  if (dryRun) {
    console.log(`[dry-run] would copy ${from} -> ${to}`);
    return;
  }
  await mkdir(dirname(to), { recursive: true });
  await cp(from, to, { recursive: true });
  console.log(`copied ${from} -> ${to}`);
}

async function installCursor() {
  const source = join(here, 'hosts', 'cursor', 'rocky-mem.mdc');
  const body = await readFile(source, 'utf8');
  await put(join(cwd, '.cursor', 'rules', 'rocky-mem.mdc'), body);
}

async function installClaudeCode() {
  const dir = join(homedir(), '.claude', 'skills', 'rocky-mem');
  await copy(join(here, 'SKILL.md'), join(dir, 'SKILL.md'));
  await copy(join(here, 'references'), join(dir, 'references'));
}

async function installDsh() {
  const dir = join(homedir(), '.dsh', 'skills', 'rocky-mem');
  await copy(join(here, 'SKILL.md'), join(dir, 'SKILL.md'));
  await copy(join(here, 'references'), join(dir, 'references'));
  console.log('restart dsh (or start a new session) to pick the skill up');
}

const START = '<!-- rocky-mem:start -->';
const END = '<!-- rocky-mem:end -->';

async function installCodex() {
  const snippet = await readFile(join(here, 'hosts', 'codex', 'AGENTS.md.snippet'), 'utf8').catch(
    () => `${START}\nsee https://github.com/linyu19872008/rocky-mem\n${END}\n`,
  );
  const target = join(cwd, 'AGENTS.md');
  const previous = (await exists(target)) ? await readFile(target, 'utf8') : '';
  const start = previous.indexOf(START);
  const end = previous.indexOf(END);
  let next;
  if (start >= 0 && end > start) {
    next = previous.slice(0, start) + snippet.trim() + previous.slice(end + END.length);
  } else {
    next = `${previous.trimEnd()}\n\n${snippet.trim()}\n`;
  }
  await put(target, next);
}

async function main() {
  if (command === '--help' || command === 'help') {
    console.log('usage: rocky-mem-skill install --host <cursor|claude-code|codex|dsh> [--cwd PATH] [--dry-run]');
    return;
  }
  if (command !== 'install') {
    console.error(`unknown command: ${command}`);
    process.exit(1);
  }
  if (!HOSTS.includes(host)) {
    console.error(`--host must be one of: ${HOSTS.join(', ')}`);
    process.exit(1);
  }
  if (host === 'cursor') await installCursor();
  else if (host === 'claude-code') await installClaudeCode();
  else if (host === 'dsh') await installDsh();
  else await installCodex();
}

await main();
