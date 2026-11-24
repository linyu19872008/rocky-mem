// Package redact applies a minimal P0a rule pack before persist.
package redact

import (
	"regexp"
	"strings"
)

const RulePackVersion = "p0a.1"

var rules = []struct {
	name string
	re   *regexp.Regexp
	repl string
}{
	{"aws_access_key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`), "[REDACTED:aws_key]"},
	{"generic_secret", regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password)\s*[:=]\s*['"]?[^\s'"]{8,}`), "${1}=[REDACTED]"},
	{"pem_block", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), "[REDACTED:pem]"},
	{"bearer", regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`), "Bearer [REDACTED]"},
}

// Result of a redaction pass.
type Result struct {
	Text       string
	Hits       map[string]int
	Level      int  // 1..3; 4 means reject
	Rejected   bool // L4: never persist
	RejectWhy  string
	RulePack   string
}

// Apply runs all rules. Language detection is ignored: every rule always runs.
func Apply(text string) Result {
	out := text
	hits := map[string]int{}
	for _, r := range rules {
		n := 0
		out = r.re.ReplaceAllStringFunc(out, func(m string) string {
			n++
			return r.re.ReplaceAllString(m, r.repl)
		})
		if n > 0 {
			hits[r.name] = n
		}
	}
	level := 2
	if len(hits) > 0 {
		level = 3
	}
	// Heuristic L4: still looks like a live private key after redaction.
	if strings.Contains(out, "PRIVATE KEY-----") {
		return Result{
			Text: out, Hits: hits, Level: 4, Rejected: true,
			RejectWhy: "private key material survived redaction",
			RulePack:  RulePackVersion,
		}
	}
	return Result{Text: out, Hits: hits, Level: level, RulePack: RulePackVersion}
}
