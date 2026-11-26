package daemon

// A daemon registry points every local client at the daemon that is actually
// running, regardless of which DSN started it.
//
// Discovery files live next to the database, so `rockymem mcp` with no --dsn
// used to look beside the *default* database and open a second, empty store
// while a daemon served another one. The registry file lives in the rockymem
// home and records the live daemon's URL, token, and DSN, so a client without
// an explicit DSN proxies to the real store instead of quietly opening a
// different one.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/config"
)

const registryFile = "daemon.json"

// Registry is the live daemon descriptor.
type Registry struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	DSN   string `json:"dsn"`
	PID   int    `json:"pid"`
	At    int64  `json:"at"`
}

func registryPath() (string, error) {
	home, err := config.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, registryFile), nil
}

// WriteRegistry publishes the running daemon for local clients.
func WriteRegistry(url, token, dsn string) error {
	p, err := registryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(Registry{
		URL:   url,
		Token: token,
		DSN:   dsn,
		PID:   os.Getpid(),
		At:    time.Now().Unix(),
	})
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// ClearRegistry removes the descriptor; a stale one is harmless because a
// failed dial falls back to the local store.
func ClearRegistry() {
	if p, err := registryPath(); err == nil {
		_ = os.Remove(p)
	}
}

// ReadRegistry returns the descriptor written by the running daemon.
func ReadRegistry() (Registry, bool) {
	p, err := registryPath()
	if err != nil {
		return Registry{}, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Registry{}, false
	}
	var r Registry
	if err := json.Unmarshal(b, &r); err != nil || r.URL == "" || r.Token == "" {
		return Registry{}, false
	}
	return r, true
}
