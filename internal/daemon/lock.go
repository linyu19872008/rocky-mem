package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// Lock is an exclusive flock on the store lockfile.
type Lock struct {
	path string
	file *os.File
}

// Acquire takes a non-blocking exclusive lock. Callers must Unlock.
func Acquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("store locked (is `rockymem serve` running?): %w", err)
	}
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	return &Lock{path: path, file: f}, nil
}

// Unlock releases the flock and closes the file.
func (l *Lock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	return err
}
