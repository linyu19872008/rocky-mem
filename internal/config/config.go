// Package config resolves state directory and DSN defaults.
package config

import (
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

// ParseDSN accepts sqlite://path, postgres://… / postgresql://…, or a bare path.
// For postgres drivers, rest is the full DSN (including scheme).
func ParseDSN(dsn string) (driver, rest string, err error) {
	if dsn == "" {
		p, err := DefaultSQLitePath()
		if err != nil {
			return "", "", err
		}
		return "sqlite", p, nil
	}
	const sqlitePrefix = "sqlite://"
	if len(dsn) >= len(sqlitePrefix) && dsn[:len(sqlitePrefix)] == sqlitePrefix {
		return "sqlite", dsn[len(sqlitePrefix):], nil
	}
	if len(dsn) >= 11 && dsn[:11] == "postgres://" {
		return "postgres", dsn, nil
	}
	if len(dsn) >= 14 && dsn[:14] == "postgresql://" {
		return "postgres", dsn, nil
	}
	// bare path
	return "sqlite", dsn, nil
}
