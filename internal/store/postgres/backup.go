package postgres

import (
	"context"
	"fmt"
)

// Backup is not implemented for PostgreSQL in this slice — use pg_dump / PITR.
// The method exists to satisfy store.Store; callers get a clear error.
func (db *DB) Backup(ctx context.Context, destPath string) error {
	_ = ctx
	_ = destPath
	return fmt.Errorf("postgres backup: use pg_dump or a managed snapshot; VACUUM INTO is SQLite-only")
}
