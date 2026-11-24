// Package daemon manages solo-mode process exclusivity and discovery files.
package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func NewToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

func ClearDiscovery(p Paths) {
	_ = os.Remove(p.Addr)
	_ = os.Remove(p.Token)
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
func ResolveURL(dbPath string) (baseURL, token string, ok bool) {
	if u := strings.TrimSpace(os.Getenv(EnvURL)); u != "" {
		tok := strings.TrimSpace(os.Getenv(EnvToken))
		return strings.TrimRight(u, "/"), tok, true
	}
	p := PathsForDB(dbPath)
	addr, tok, err := ReadDiscovery(p)
	if err != nil {
		return "", "", false
	}
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/"), tok, true
}
