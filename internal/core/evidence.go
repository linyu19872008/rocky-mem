package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// resolveEvidenceText turns evidence into persistable body text.
// commit/path do best-effort shallow resolution against cwd; on failure the
// caller still stores a textual fallback and may emit a warning.
func resolveEvidenceText(evType, ref, text, cwd string) (body string, warning string) {
	text = strings.TrimSpace(text)
	ref = strings.TrimSpace(ref)
	switch evType {
	case "quote":
		return text, ""
	case "commit":
		if cwd != "" && ref != "" {
			if out, err := gitShowCommit(cwd, ref); err == nil && strings.TrimSpace(out) != "" {
				return strings.TrimSpace(out), ""
			}
			warning = "commit not resolved via git; stored ref text"
		}
		if text != "" {
			return text, warning
		}
		if ref != "" {
			return "commit " + ref, warning
		}
		return "", "commit evidence missing ref/text"
	case "path":
		if cwd != "" && ref != "" {
			if out, err := pathStatSummary(cwd, ref); err == nil {
				return out, ""
			}
			warning = "path not found under cwd; stored ref text"
		}
		if text != "" {
			return text, warning
		}
		if ref != "" {
			return "path " + ref, warning
		}
		return "", "path evidence missing ref/text"
	case "url":
		if text != "" {
			return text, ""
		}
		return ref, ""
	case "session":
		if text != "" {
			return text, ""
		}
		if ref != "" {
			return "session " + ref, ""
		}
		return "", "session evidence missing ref/text"
	default:
		if text != "" {
			return text, "unknown evidence type " + evType
		}
		return ref, "unknown evidence type " + evType
	}
}

func gitShowCommit(cwd, ref string) (string, error) {
	cmd := exec.Command("git", "-C", cwd, "log", "-1", "--format=%H%n%s%n%an <%ae>%n%ci", ref)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return strings.TrimSpace(string(out)), nil
	}
	sha := lines[0]
	subject := lines[1]
	author := ""
	when := ""
	if len(lines) > 2 {
		author = lines[2]
	}
	if len(lines) > 3 {
		when = lines[3]
	}
	return fmt.Sprintf("commit %s\n%s\n%s\n%s", shortSHA(sha), subject, author, when), nil
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

func pathStatSummary(cwd, ref string) (string, error) {
	p := ref
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, ref)
	}
	fi, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	rel := ref
	if absCwd, err := filepath.Abs(cwd); err == nil {
		if absP, err := filepath.Abs(p); err == nil {
			if r, err := filepath.Rel(absCwd, absP); err == nil {
				rel = r
			}
		}
	}
	mode := "file"
	if fi.IsDir() {
		mode = "dir"
	}
	return fmt.Sprintf("path %s\nmode=%s size=%d mtime=%s",
		rel, mode, fi.Size(), fi.ModTime().UTC().Format(time.RFC3339)), nil
}
