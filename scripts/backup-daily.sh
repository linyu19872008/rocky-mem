#!/usr/bin/env bash
# Daily (or cron) snapshot of the local rocky-mem store.
# Safe while `make serve` is running: CLI prefers the daemon HTTP API.
#
#   ./scripts/backup-daily.sh
#   RETAIN=30 ./scripts/backup-daily.sh
#   crontab -e  →  15 3 * * * cd /path/to/rocky-mem && ./scripts/backup-daily.sh >>.local/backup.log 2>&1
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${ROOT}/bin/rockymem"
DSN="${ROCKYMEM_DSN:-sqlite://${ROOT}/.local/rocky-mem.db}"
export ROCKYMEM_HOME="${ROCKYMEM_HOME:-${ROOT}/.local}"
RETAIN="${RETAIN:-14}"
OUT_DIR="${ROCKYMEM_BACKUP_DIR:-${ROCKYMEM_HOME}/backups}"

mkdir -p "${OUT_DIR}"
make -C "${ROOT}" build >/dev/null
"${BIN}" --dsn "${DSN}" backup --dir "${OUT_DIR}" --retain "${RETAIN}"
echo "backup ok $(date -u +%Y-%m-%dT%H:%M:%SZ) dir=${OUT_DIR} retain=${RETAIN}"
