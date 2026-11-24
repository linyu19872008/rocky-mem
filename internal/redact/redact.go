// Package redact applies a minimal P0a rule pack before persist.
package redact

import (
	"regexp"
	"sort"
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

// Segment is one piece of an inline redaction diff (eq / del / ins).
type Segment struct {
	Kind string `json:"kind"` // eq | del | ins
	Text string `json:"text"`
	Rule string `json:"rule,omitempty"`
}

// Span records one replacement against the original text (byte offsets).
type Span struct {
	Rule        string `json:"rule"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}

// Result of a redaction pass.
type Result struct {
	Text       string
	Hits       map[string]int
	Spans      []Span
	Segments   []Segment
	Level      int  // 1..3; 4 means reject
	Rejected   bool // L4: never persist
	RejectWhy  string
	RulePack   string
	Unchanged  bool
}

type match struct {
	start, end int
	rule, repl string
	original   string
}

// Apply runs all rules. Language detection is ignored: every rule always runs.
func Apply(text string) Result {
	var matches []match
	hits := map[string]int{}
	for _, r := range rules {
		locs := r.re.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			m := text[loc[0]:loc[1]]
			repl := r.re.ReplaceAllString(m, r.repl)
			matches = append(matches, match{
				start: loc[0], end: loc[1], rule: r.name, repl: repl, original: m,
			})
			hits[r.name]++
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start == matches[j].start {
			return matches[i].end > matches[j].end
		}
		return matches[i].start < matches[j].start
	})
	// Drop overlaps (keep earliest / longest-first already sorted).
	var kept []match
	lastEnd := 0
	for _, m := range matches {
		if m.start < lastEnd {
			hits[m.rule]--
			if hits[m.rule] <= 0 {
				delete(hits, m.rule)
			}
			continue
		}
		kept = append(kept, m)
		lastEnd = m.end
	}

	var spans []Span
	var segs []Segment
	var out strings.Builder
	pos := 0
	for _, m := range kept {
		if m.start > pos {
			eq := text[pos:m.start]
			out.WriteString(eq)
			segs = append(segs, Segment{Kind: "eq", Text: eq})
		}
		out.WriteString(m.repl)
		spans = append(spans, Span{
			Rule: m.rule, Start: m.start, End: m.end,
			Original: m.original, Replacement: m.repl,
		})
		segs = append(segs,
			Segment{Kind: "del", Text: m.original, Rule: m.rule},
			Segment{Kind: "ins", Text: m.repl, Rule: m.rule},
		)
		pos = m.end
	}
	if pos < len(text) {
		eq := text[pos:]
		out.WriteString(eq)
		segs = append(segs, Segment{Kind: "eq", Text: eq})
	}
	if len(segs) == 0 {
		segs = []Segment{{Kind: "eq", Text: text}}
	}

	level := 2
	if len(hits) > 0 {
		level = 3
	}
	resText := out.String()
	if len(kept) == 0 {
		resText = text
	}
	rejected := false
	rejectWhy := ""
	if strings.Contains(resText, "PRIVATE KEY-----") {
		rejected = true
		rejectWhy = "private key material survived redaction"
		level = 4
	}
	return Result{
		Text: resText, Hits: hits, Spans: spans, Segments: segs,
		Level: level, Rejected: rejected, RejectWhy: rejectWhy,
		RulePack: RulePackVersion, Unchanged: len(kept) == 0,
	}
}
