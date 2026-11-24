# Backups (solo SQLite)

Snapshots use SQLite `VACUUM INTO` — a consistent, compacted copy.

## One-shot

```bash
# while serve may be running (CLI talks HTTP to the daemon)
make backup
# or:
./bin/rockymem --dsn sqlite://$PWD/.local/rocky-mem.db backup --dir .local/backups --retain 14
```

## Daily automation

```bash
./scripts/backup-daily.sh
# RETAIN=30 ROCKYMEM_BACKUP_DIR=$PWD/.local/backups ./scripts/backup-daily.sh
```

### macOS LaunchAgent (recommended on this machine)

```bash
./scripts/install-backup-launchd.sh
# runs daily at 03:15 local; logs to .local/backup.log
```

If LaunchAgent logs `Operation not permitted` (common when the repo lives under
Desktop/Documents), grant **Full Disk Access** to `/bin/bash` or move the DB under
`~/Library/...`, or keep using cron/`./scripts/backup-daily.sh` from an interactive shell.

### Cron

```
15 3 * * * cd /path/to/rocky-mem && ./scripts/backup-daily.sh >> .local/backup.log 2>&1
```

## Restore

Stop `serve` first (discovery files must be absent), then:

```bash
./bin/rockymem --dsn sqlite://$PWD/.local/rocky-mem.db restore .local/backups/rocky-mem-YYYYMMDD-HHMMSS.db
```

## Retention

`--retain N` keeps the newest N files matching `rocky-mem-*.db` in the backup directory.
`RETAIN` in `backup-daily.sh` defaults to 14.
