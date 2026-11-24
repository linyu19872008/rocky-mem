#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const en = JSON.parse(readFileSync(join(root, "locales/en.json"), "utf8"));
const zh = JSON.parse(readFileSync(join(root, "locales/zh-CN.json"), "utf8"));

function flatten(obj, prefix = "", out = {}) {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === "object" && !Array.isArray(v)) flatten(v, key, out);
    else out[key] = v;
  }
  return out;
}

const enFlat = flatten(en);
const zhFlat = flatten(zh);
const enKeys = Object.keys(enFlat).sort();
const zhKeys = Object.keys(zhFlat).sort();
const missingInZh = enKeys.filter((k) => !(k in zhFlat));
const missingInEn = zhKeys.filter((k) => !(k in enFlat));
const empty = [...enKeys, ...zhKeys].filter((k) => {
  const v = enFlat[k] ?? zhFlat[k];
  return typeof v === "string" && v.trim() === "";
});

let failed = false;
if (missingInZh.length) {
  console.error("Missing in zh-CN:", missingInZh.join(", "));
  failed = true;
}
if (missingInEn.length) {
  console.error("Missing in en:", missingInEn.join(", "));
  failed = true;
}
if (empty.length) {
  console.error("Empty strings:", [...new Set(empty)].join(", "));
  failed = true;
}
if (failed) process.exit(1);
console.log(`OK: i18n parity (${enKeys.length} keys)`);
