// Package termmap expands bilingual queries using the curated seed map.
package termmap

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/linyu19872008/rocky-mem/data"
	"gopkg.in/yaml.v3"
)

const DefaultWeight = 0.7

// Map is an in-memory zh→en expansion table.
type Map struct {
	byZH map[string][]string // lowercased zh phrase → en terms
}

type fileShape struct {
	Entries []struct {
		ZH string   `yaml:"zh"`
		EN []string `yaml:"en"`
	} `yaml:"entries"`
}

var (
	defaultOnce sync.Once
	defaultMap  *Map
	defaultErr  error
)

// LoadDefault loads the embedded seed term map.
func LoadDefault() (*Map, error) {
	defaultOnce.Do(func() {
		defaultMap, defaultErr = Parse(data.TermMapYAML)
	})
	return defaultMap, defaultErr
}

// Parse builds a Map from YAML bytes.
func Parse(b []byte) (*Map, error) {
	var f fileShape
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	m := &Map{byZH: make(map[string][]string, len(f.Entries))}
	for _, e := range f.Entries {
		zh := strings.TrimSpace(e.ZH)
		if zh == "" || len(e.EN) == 0 {
			continue
		}
		key := strings.ToLower(zh)
		seen := map[string]struct{}{}
		var ens []string
		for _, en := range e.EN {
			en = strings.TrimSpace(en)
			if en == "" {
				continue
			}
			lk := strings.ToLower(en)
			if _, ok := seen[lk]; ok {
				continue
			}
			seen[lk] = struct{}{}
			ens = append(ens, en)
		}
		if len(ens) > 0 {
			m.byZH[key] = ens
		}
	}
	return m, nil
}

// Expanded is one search variant with a relative weight.
type Expanded struct {
	Query  string
	Weight float64 // 1.0 = original; DefaultWeight for bridged terms
}

// Expand returns the original query plus English expansions when zh phrases match.
// Longest phrase match wins per occurrence; expansions are de-duplicated.
func (m *Map) Expand(query string) []Expanded {
	q := strings.TrimSpace(query)
	out := []Expanded{{Query: q, Weight: 1.0}}
	if m == nil || q == "" {
		return out
	}
	lower := strings.ToLower(q)
	seenQ := map[string]struct{}{lower: {}}

	// Prefer longer zh keys first.
	keys := make([]string, 0, len(m.byZH))
	for k := range m.byZH {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if utf8.RuneCountInString(keys[j]) > utf8.RuneCountInString(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, zh := range keys {
		if !strings.Contains(lower, zh) {
			continue
		}
		for _, en := range m.byZH[zh] {
			el := strings.ToLower(en)
			if _, ok := seenQ[el]; ok {
				continue
			}
			seenQ[el] = struct{}{}
			out = append(out, Expanded{Query: en, Weight: DefaultWeight})
		}
	}
	return out
}
