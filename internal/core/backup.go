package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/errs"
)

// BackupResult summarises a snapshot.
type BackupResult struct {
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	CreatedAt string `json:"created_at"`
}

// Backup writes VACUUM INTO dest (or default timestamped path under dir).
func (s *Service) Backup(ctx context.Context, dest string) (*BackupResult, error) {
	if dest == "" {
		return nil, errs.New(errs.FieldRequired, "backup path required", nil)
	}
	if err := s.Store.Backup(ctx, dest); err != nil {
		return nil, err
	}
	fi, err := os.Stat(dest)
	if err != nil {
		return nil, err
	}
	_ = s.Store.InsertAudit(ctx, "backup", "store", nil, "", dest)
	return &BackupResult{
		Path: dest, Bytes: fi.Size(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// DefaultBackupPath returns <dir>/rocky-mem-YYYYMMDD-HHMMSS.db
func DefaultBackupPath(dir string) string {
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, fmt.Sprintf("rocky-mem-%s.db", time.Now().UTC().Format("20060102-150405")))
}

// RestoreFile copies a snapshot over destPath. Caller must not hold the DB open.
func RestoreFile(src, dest string) error {
	if src == "" || dest == "" {
		return errs.New(errs.FieldRequired, "src and dest required", nil)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	tmp := dest + ".restoring"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}
