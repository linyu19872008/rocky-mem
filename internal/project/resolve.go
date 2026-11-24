// Package project resolves cwd → git remote → normalised project identity.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reHTTPS = regexp.MustCompile(`(?i)^https?://([^/@]+@)?([^/]+)/(.+?)(?:\.git)?/?$`)
	reSSH   = regexp.MustCompile(`(?i)^(?:ssh://)?(?:[^@]+@)?([^/:]+)[:/](.+?)(?:\.git)?/?$`)
	reSCP   = regexp.MustCompile(`(?i)^git@([^:]+):(.+?)(?:\.git)?/?$`)
)

// ResolveResult is the identity of a working directory.
type ResolveResult struct {
	Name          string
	RemoteURLNorm string
	RootHint      string
	LocalOnly     bool // no remote; identity is path hash
}

// NormalizeRemote turns various git remote URL forms into host/owner/repo.
// Host is lowercased; owner/repo case is preserved.
func NormalizeRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if m := reHTTPS.FindStringSubmatch(raw); m != nil {
		host := strings.ToLower(m[2])
		path := strings.TrimSuffix(m[3], "/")
		path = strings.TrimSuffix(path, ".git")
		return host + "/" + path
	}
	if m := reSCP.FindStringSubmatch(raw); m != nil {
		host := strings.ToLower(m[1])
		path := strings.TrimSuffix(m[2], "/")
		path = strings.TrimSuffix(path, ".git")
		return host + "/" + path
	}
	if m := reSSH.FindStringSubmatch(raw); m != nil {
		host := strings.ToLower(m[1])
		path := strings.TrimSuffix(m[2], "/")
		path = strings.TrimSuffix(path, ".git")
		return host + "/" + path
	}
	return ""
}

// Resolve inspects cwd for a git remote. Offline repos without remotes
// fall back to a hash of the absolute path (documented degradation).
func Resolve(cwd string) (*ResolveResult, error) {
	if cwd == "" {
		return nil, nil
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, err
	}

	root := gitRoot(abs)
	if root == "" {
		// Not a git repo: local-only project keyed by path hash.
		sum := sha256.Sum256([]byte(abs))
		return &ResolveResult{
			Name:          filepath.Base(abs),
			RemoteURLNorm: "local/" + hex.EncodeToString(sum[:8]),
			RootHint:      abs,
			LocalOnly:     true,
		}, nil
	}

	remote := gitRemote(root)
	norm := NormalizeRemote(remote)
	name := filepath.Base(root)
	if norm != "" {
		parts := strings.Split(norm, "/")
		if len(parts) >= 2 {
			name = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
	} else {
		sum := sha256.Sum256([]byte(root))
		norm = "local/" + hex.EncodeToString(sum[:8])
	}

	return &ResolveResult{
		Name:          name,
		RemoteURLNorm: norm,
		RootHint:      root,
		LocalOnly:     strings.HasPrefix(norm, "local/"),
	}, nil
}

func gitRoot(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitRemote(root string) string {
	cmd := exec.Command("git", "-C", root, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
