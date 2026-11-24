#!/usr/bin/env bash
# Install a daily LaunchAgent for scripts/backup-daily.sh (macOS).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LABEL="com.rocky-mem.backup"
DEST="${HOME}/Library/LaunchAgents/${LABEL}.plist"
TEMPLATE="${ROOT}/scripts/${LABEL}.plist.template"

mkdir -p "${HOME}/Library/LaunchAgents" "${ROOT}/.local"
sed "s|__ROOT__|${ROOT}|g" "${TEMPLATE}" > "${DEST}"
launchctl bootout "gui/$(id -u)/${LABEL}" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "${DEST}"
launchctl enable "gui/$(id -u)/${LABEL}" 2>/dev/null || true
# Kick once so the agent is registered and log path exists.
launchctl kickstart -k "gui/$(id -u)/${LABEL}" 2>/dev/null || launchctl start "${LABEL}" || true
echo "installed ${DEST}"
echo "schedule: daily 03:15 local; log: ${ROOT}/.local/backup.log"
echo "unload: launchctl bootout gui/$(id -u)/${LABEL}"
