package daemon_test

import (
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/daemon"
)

func TestLockExclusive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rocky-mem.lock")
	l1, err := daemon.Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Unlock()

	if _, err := daemon.Acquire(path); err == nil {
		t.Fatal("expected second acquire to fail")
	}
	if err := l1.Unlock(); err != nil {
		t.Fatal(err)
	}
	l2, err := daemon.Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = l2.Unlock()
}

func TestPathsForDB(t *testing.T) {
	p := daemon.PathsForDB("/tmp/x/rocky-mem.db")
	if p.Lock != "/tmp/x/rocky-mem.lock" || p.Addr != "/tmp/x/rocky-mem.addr" {
		t.Fatalf("%+v", p)
	}
}
