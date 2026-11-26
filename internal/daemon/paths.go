// Package daemon manages solo-mode process exclusivity and discovery files.
package daemon

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/config"
)

const (
	DefaultBind = "127.0.0.1:7432"
	EnvURL      = "ROCKYMEM_URL"
	EnvToken    = "ROCKYMEM_TOKEN"
)

// Paths for a given sqlite database file.
type Paths struct {
	DB    string
	Lock  string
	Addr  string
	Token string
}

func PathsForDB(dbPath string) Paths {
	dir := filepath.Dir(dbPath)
	base := filepath.Base(dbPath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" {
		stem = "rocky-mem"
	}
	return Paths{
		DB:    dbPath,
		Lock:  filepath.Join(dir, stem+".lock"),
		Addr:  filepath.Join(dir, stem+".addr"),
		Token: filepath.Join(dir, stem+".token"),
	}
}

// PathsForPostgres places discovery files under $ROCKYMEM_HOME/pg-<sha256prefix>/
// derived from the DSN so serve/MCP can share addr+token without a DB file path.
func PathsForPostgres(dsn string) (Paths, error) {
	home, err := config.Home()
	if err != nil {
		return Paths{}, err
	}
	sum := sha256.Sum256([]byte(dsn))
	dir := filepath.Join(home, fmt.Sprintf("pg-%x", sum[:8]))
	return Paths{
		DB:    dsn,
		Lock:  filepath.Join(dir, "rocky-mem.lock"),
		Addr:  filepath.Join(dir, "rocky-mem.addr"),
		Token: filepath.Join(dir, "rocky-mem.token"),
	}, nil
}

// PathsForDriver selects SQLite file-adjacent paths or a hashed postgres discovery dir.
func PathsForDriver(driver, rest string) (Paths, error) {
	switch driver {
	case "postgres":
		return PathsForPostgres(rest)
	default:
		return PathsForDB(rest), nil
	}
}

// ResolveURLForDriver is ResolveURL using dialect-aware discovery paths.
func ResolveURLForDriver(driver, rest string) (baseURL, token string, ok bool) {
	p, err := PathsForDriver(driver, rest)
	if err != nil {
		return "", "", false
	}
	return resolveURLPaths(p)
}

func NewToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LoadOrCreateToken reuses a non-empty token file so serve restarts keep the
// same bearer for long-lived MCP/CLI clients (Cursor keeps stdio mcp alive).
// created is true only when a new token was minted (missing or empty file).
func LoadOrCreateToken(tokenPath string) (token string, created bool, err error) {
	if b, err := os.ReadFile(tokenPath); err == nil {
		if tok := strings.TrimSpace(string(b)); tok != "" {
			return tok, false, nil
		}
	}
	tok, err := NewToken()
	if err != nil {
		return "", false, err
	}
	return tok, true, nil
}

func WriteDiscovery(p Paths, addr, token string) error {
	if err := os.MkdirAll(filepath.Dir(p.Addr), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p.Addr, []byte(addr+"\n"), 0o600); err != nil {
		return err
	}
	return os.WriteFile(p.Token, []byte(token+"\n"), 0o600)
}

// ClearDiscovery removes the live address file. The token file is kept so the
// next serve can reuse it (see LoadOrCreateToken).
func ClearDiscovery(p Paths) {
	_ = os.Remove(p.Addr)
}

func ReadDiscovery(p Paths) (addr, token string, err error) {
	ab, err := os.ReadFile(p.Addr)
	if err != nil {
		return "", "", err
	}
	tb, err := os.ReadFile(p.Token)
	if err != nil {
		return "", "", err
	}
	addr = strings.TrimSpace(string(ab))
	token = strings.TrimSpace(string(tb))
	if addr == "" || token == "" {
		return "", "", fmt.Errorf("empty discovery files")
	}
	return addr, token, nil
}

// ResolveURL returns base URL for API client: env wins, else discovery file.
// Env URL without a token (env or token file) is treated as unresolved so
// callers do not dial with an empty bearer.
func ResolveURL(dbPath string) (baseURL, token string, ok bool) {
	return resolveURLPaths(PathsForDB(dbPath))
}

func resolveURLPaths(p Paths) (baseURL, token string, ok bool) {
	if u := strings.TrimSpace(os.Getenv(EnvURL)); u != "" {
		tok := strings.TrimSpace(os.Getenv(EnvToken))
		if tok == "" {
			if tb, err := os.ReadFile(p.Token); err == nil {
				tok = strings.TrimSpace(string(tb))
			}
		}
		if tok == "" {
			return "", "", false
		}
		return strings.TrimRight(u, "/"), tok, true
	}
	addr, tok, err := ReadDiscovery(p)
	if err != nil {
		return "", "", false
	}
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/"), tok, true
}
