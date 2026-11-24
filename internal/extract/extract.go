package extract

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

const (
	MaxBytes     = 15 << 20 // 15 MiB raw upload
	MaxTextRunes = 200_000
)

// Result is extracted plain text ready for redaction.
type Result struct {
	Format      Format
	Text        string
	ContentHash string // sha256 of raw bytes
	CharCount   int
	PagesHint   int // pdf pages if known; else 0
}

// FromBytes detects format and extracts UTF-8 text.
func FromBytes(filename string, data []byte) (*Result, error) {
	if len(data) == 0 {
		return nil, unsupported("empty file")
	}
	if len(data) > MaxBytes {
		return nil, unsupported("file too large (max 15MB)")
	}
	format, err := Detect(filename, data)
	if err != nil {
		return nil, err
	}
	var text string
	switch format {
	case FormatPlain:
		text, err = extractPlain(data)
	case FormatDOCX:
		text, err = extractDOCX(data)
	case FormatXLSX:
		text, err = extractXLSX(data)
	case FormatPDF:
		text, err = extractPDF(data)
	default:
		err = unsupported("unknown format")
	}
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, unsupported("extracted text is empty")
	}
	if utf8.RuneCountInString(text) > MaxTextRunes {
		runes := []rune(text)
		text = string(runes[:MaxTextRunes]) + "\n…[truncated]"
	}
	sum := sha256.Sum256(data)
	return &Result{
		Format:      format,
		Text:        text,
		ContentHash: hex.EncodeToString(sum[:]),
		CharCount:   utf8.RuneCountInString(text),
	}, nil
}

// Chunk splits text into roughly size-limited blocks on paragraph boundaries.
func Chunk(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 3500
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	paras := strings.Split(text, "\n\n")
	var out []string
	var cur strings.Builder
	curRunes := 0
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			out = append(out, s)
		}
		cur.Reset()
		curRunes = 0
	}
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		pr := utf8.RuneCountInString(p)
		if curRunes > 0 && curRunes+pr+2 > maxRunes {
			flush()
		}
		if pr > maxRunes {
			flush()
			rs := []rune(p)
			for len(rs) > 0 {
				n := maxRunes
				if n > len(rs) {
					n = len(rs)
				}
				out = append(out, string(rs[:n]))
				rs = rs[n:]
			}
			continue
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
			curRunes += 2
		}
		cur.WriteString(p)
		curRunes += pr
	}
	flush()
	return out
}
