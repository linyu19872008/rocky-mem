// Package config resolves state directory and DSN defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const EnvHome = "ROCKYMEM_HOME"

// Home returns the rocky-mem state directory.
func Home() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return filepath.Clean(v), nil
	}
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "rocky-mem"), nil
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "rocky-mem"), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "rocky-mem"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "rocky-mem"), nil
	}
}

// DefaultSQLitePath is Home()/rocky-mem.db.
func DefaultSQLitePath() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "rocky-mem.db"), nil
}

// ParseDSN accepts sqlite://path or a bare filesystem path.
// Postgres is reserved for a later phase; P0a only opens SQLite.
func ParseDSN(dsn string) (driver, path string, err error) {
	if dsn == "" {
		p, err := DefaultSQLitePath()
		if err != nil {
			return "", "", err
		}
		return "sqlite", p, nil
	}
	const prefix = "sqlite://"
	if len(dsn) >= len(prefix) && dsn[:len(prefix)] == prefix {
		return "sqlite", dsn[len(prefix):], nil
	}
	if len(dsn) >= 11 && dsn[:11] == "postgres://" || len(dsn) >= 14 && dsn[:14] == "postgresql://" {
		return "", "", fmt.Errorf("postgres backend is not implemented in P0a; use sqlite://…")
	}
	// bare path
	return "sqlite", dsn, nil
}
