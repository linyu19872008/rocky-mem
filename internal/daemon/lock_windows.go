//go:build windows

package daemon

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFile takes an exclusive, fail-fast lock on the first byte. Windows has no
// flock; LockFileEx is the equivalent guarantee, and the OS releases it when the
// process exits.
func lockFile(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &overlapped,
	)
}

func unlockFile(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlapped)
}
