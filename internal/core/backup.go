package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/errs"
)

// BackupResult summarises a snapshot.
type BackupResult struct {
	Path       string   `json:"path"`
	Bytes      int64    `json:"bytes"`
	CreatedAt  string   `json:"created_at"`
	Pruned     []string `json:"pruned,omitempty"`
	RetainKeep int      `json:"retain_keep,omitempty"`
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

// BackupWithRetain writes a timestamped snapshot under dir and keeps the newest keep files.
// keep <= 0 means retain forever (no prune).
func (s *Service) BackupWithRetain(ctx context.Context, dir string, keep int) (*BackupResult, error) {
	if dir == "" {
		return nil, errs.New(errs.FieldRequired, "backup dir required", nil)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	dest := DefaultBackupPath(dir)
	out, err := s.Backup(ctx, dest)
	if err != nil {
		return nil, err
	}
	if keep > 0 {
		pruned, err := PruneBackups(dir, keep)
		if err != nil {
			return out, err
		}
		out.Pruned = pruned
		out.RetainKeep = keep
	}
	return out, nil
}

// DefaultBackupPath returns <dir>/rocky-mem-YYYYMMDD-HHMMSS.db
func DefaultBackupPath(dir string) string {
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, fmt.Sprintf("rocky-mem-%s.db", time.Now().UTC().Format("20060102-150405")))
}

// PruneBackups keeps the newest keep files matching rocky-mem-*.db in dir.
func PruneBackups(dir string, keep int) ([]string, error) {
	if keep <= 0 {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type item struct {
		name string
		mod  time.Time
	}
	var files []item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "rocky-mem-") || !strings.HasSuffix(name, ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, item{name: name, mod: info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	var pruned []string
	for i := keep; i < len(files); i++ {
		path := filepath.Join(dir, files[i].name)
		if err := os.Remove(path); err != nil {
			return pruned, err
		}
		pruned = append(pruned, path)
	}
	return pruned, nil
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
