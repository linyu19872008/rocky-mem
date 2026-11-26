#!/usr/bin/env node
// Copy the canonical skill from ../skills/rocky-mem into this package.
// Runs at publish time (prepack) so the npm package and the repository can
// never disagree; `npm run prepack` locally to refresh the committed copies.

import { cp, readdir, rm } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const source = join(here, '..', '..', 'skills', 'rocky-mem');
const targets = ['SKILL.md', 'references', 'hosts', 'tools.json'];

for (const name of targets) {
  const from = join(source, name);
  const to = join(here, name);
  const isDir = name === 'references' || name === 'hosts';
  if (isDir) {
    await rm(to, { recursive: true, force: true });
    await cp(from, to, { recursive: true });
  } else {
    await cp(from, to, { force: true });
  }
}

const entries = await readdir(here);
console.log(`synced ${targets.join(', ')} into ${here}`);
console.log(`package now holds: ${entries.filter((e) => !e.startsWith('.')).join(', ')}`);
