#!/usr/bin/env bash
# Start rockymem serve against the project .local DB (exclusive writer).
# MCP/CLI with the same --dsn discover http://addr via rocky-mem.addr + .token.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${ROOT}/bin/rockymem"
DSN="${ROCKYMEM_DSN:-sqlite://${ROOT}/.local/rocky-mem.db}"
export ROCKYMEM_HOME="${ROOT}/.local"
BIND="${ROCKYMEM_BIND:-127.0.0.1:7432}"

mkdir -p "${ROOT}/.local"
make -C "${ROOT}" build >/dev/null
exec "${BIN}" --dsn "${DSN}" serve --bind "${BIND}"
