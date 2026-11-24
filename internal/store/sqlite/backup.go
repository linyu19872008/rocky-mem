package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Backup writes a consistent snapshot via VACUUM INTO.
func (db *DB) Backup(ctx context.Context, destPath string) error {
	if destPath == "" {
		return fmt.Errorf("backup destination required")
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("backup destination already exists: %s", destPath)
	}
	// VACUUM INTO creates a new compacted DB file.
	_, err := db.sql.ExecContext(ctx, `VACUUM INTO ?`, destPath)
	return err
}
