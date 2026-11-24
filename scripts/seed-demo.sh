#!/usr/bin/env bash
# Seed a demo corpus into the local store for P0a validation.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${ROOT}/bin/rockymem"
DSN="${ROCKYMEM_DSN:-sqlite://${ROOT}/.local/rocky.db}"
export ROCKYMEM_HOME="${ROOT}/.local"

mkdir -p "${ROOT}/.local"
make -C "${ROOT}" build >/dev/null
"${BIN}" --dsn "${DSN}" init

"${BIN}" --dsn "${DSN}" remember "package manager" "Use pnpm. Never npm or yarn in this repo." \
  --key package_manager --type pref --scope global --pinned

"${BIN}" --dsn "${DSN}" record "ConfigMap hot reload lost during rollout" \
  --type pitfall \
  --symptom "config reverts after kubectl rollout restart" \
  --root-cause "file-mounted ConfigMap update lag ~60s" \
  --fix "add checksum annotation on Pod template to force recreate" \
  --applies-when "K8s + file-mounted ConfigMap + short rollout window" \
  --invalid-when "config read via ConfigMap API rather than file mount" \
  --outcome success --confidence 0.85 \
  --evidence "observed ConfigMap mount still stale after kubectl rollout restart within 30s"

"${BIN}" --dsn "${DSN}" record "Only kubectl rollout restart for ConfigMap" \
  --type howto \
  --symptom "need pods to pick up ConfigMap" \
  --fix "kubectl rollout restart deployment/x" \
  --applies-when "never for file-mounted ConfigMap during short rollouts" \
  --invalid-when "always for file-mounted ConfigMap with short windows" \
  --outcome failed --confidence 0.9 \
  --evidence "tried rollout restart alone; config still old"

"${BIN}" --dsn "${DSN}" record "SQLite FTS5 trigram for identifier search" \
  --type decision \
  --symptom "need substring match for error codes and paths" \
  --root-cause "stemming hurts exact match on ERR_ codes" \
  --fix "use FTS5 tokenize=trigram / pg_trgm" \
  --applies-when "knowledge base dominated by identifiers and error strings" \
  --invalid-when "prose-heavy corpora needing stemming" \
  --outcome success --confidence 0.8 \
  --evidence "ADR-0006 chose pg_trgm over tsvector for the same reason"

echo
echo "Demo store: ${DSN}"
echo "Try: ${BIN} --dsn ${DSN} search \"ConfigMap rollout\""
"${BIN}" --dsn "${DSN}" doctor
