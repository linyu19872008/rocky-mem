// Package textnorm applies NFC and hashing rules from the design.
package textnorm

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NFC normalises text before hash or persist.
func NFC(s string) string {
	return norm.NFC.String(s)
}

// Hash returns sha256 hex of NFC(s).
func Hash(s string) string {
	sum := sha256.Sum256([]byte(NFC(s)))
	return hex.EncodeToString(sum[:])
}

// NormalizeQuery folds whitespace and casefolds for retrieval.
func NormalizeQuery(q string) string {
	q = NFC(q)
	q = strings.ToLower(q)
	var b strings.Builder
	prevSpace := false
	for _, r := range q {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// SymptomKey derives a clustering key from symptom text.
// Language-neutral tokens (paths, error codes, tool names) preferred;
// otherwise normalize(symptom) + "#" + lang.
func SymptomKey(symptom, lang string) string {
	symptom = NFC(strings.TrimSpace(symptom))
	if symptom == "" {
		return ""
	}
	tokens := extractNeutralTokens(symptom)
	if len(tokens) > 0 {
		return Hash(strings.Join(tokens, " "))
	}
	if lang == "" {
		lang = "und"
	}
	return Hash(NormalizeQuery(symptom) + "#" + lang)
}

func extractNeutralTokens(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",;:|()[]{}\"'`", r)
	})
	var out []string
	seen := map[string]struct{}{}
	for _, f := range fields {
		if !looksNeutral(f) {
			continue
		}
		key := strings.ToLower(f)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sortStrings(out)
	return out
}

func looksNeutral(f string) bool {
	if strings.Contains(f, "/") || strings.Contains(f, "\\") || strings.Contains(f, ".") {
		if len(f) >= 3 {
			return true
		}
	}
	if strings.HasPrefix(strings.ToUpper(f), "ERR_") || strings.HasPrefix(f, "Xid") {
		return true
	}
	upper := 0
	digit := 0
	for _, r := range f {
		if unicode.IsUpper(r) {
			upper++
		}
		if unicode.IsDigit(r) {
			digit++
		}
	}
	// kubectl-like or CamelCase identifiers
	if upper >= 2 && len(f) >= 4 {
		return true
	}
	if digit > 0 && len(f) >= 4 {
		return true
	}
	return false
}

func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
