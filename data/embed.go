// Package data embeds seed assets shipped with the binary.
package data

import _ "embed"

//go:embed seed/term-map.zh-en.yaml
var TermMapYAML []byte
