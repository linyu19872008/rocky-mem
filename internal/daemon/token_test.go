package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateTokenReusesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rocky-mem.token")
	if err := os.WriteFile(path, []byte("stable-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, created, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if created || tok != "stable-token" {
		t.Fatalf("got %q created=%v", tok, created)
	}
	tok2, created2, err := LoadOrCreateToken(path)
	if err != nil || created2 || tok2 != tok {
		t.Fatalf("reuse: %q created=%v %v", tok2, created2, err)
	}
}

func TestLoadOrCreateTokenMintsWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rocky-mem.token")
	tok, created, err := LoadOrCreateToken(path)
	if err != nil || !created || tok == "" {
		t.Fatalf("mint: %q created=%v %v", tok, created, err)
	}
}

func TestClearDiscoveryKeepsToken(t *testing.T) {
	dir := t.TempDir()
	p := Paths{
		Addr:  filepath.Join(dir, "rocky-mem.addr"),
		Token: filepath.Join(dir, "rocky-mem.token"),
	}
	if err := WriteDiscovery(p, "127.0.0.1:7432", "keep-me"); err != nil {
		t.Fatal(err)
	}
	ClearDiscovery(p)
	if _, err := os.Stat(p.Addr); !os.IsNotExist(err) {
		t.Fatalf("addr should be gone: %v", err)
	}
	b, err := os.ReadFile(p.Token)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "keep-me\n" {
		t.Fatalf("token cleared: %q", b)
	}
}

func TestResolveURLRequiresToken(t *testing.T) {
	t.Setenv(EnvURL, "http://127.0.0.1:7432")
	t.Setenv(EnvToken, "")
	dir := t.TempDir()
	db := filepath.Join(dir, "rocky-mem.db")
	_, _, ok := ResolveURL(db)
	if ok {
		t.Fatal("expected unresolved without token")
	}
	if err := os.WriteFile(filepath.Join(dir, "rocky-mem.token"), []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	base, tok, ok := ResolveURL(db)
	if !ok || base != "http://127.0.0.1:7432" || tok != "from-file" {
		t.Fatalf("got %q %q ok=%v", base, tok, ok)
	}
}
