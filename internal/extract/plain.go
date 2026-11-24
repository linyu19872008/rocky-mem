package extract

import (
	"bytes"
	"unicode/utf8"
)

func extractPlain(data []byte) (string, error) {
	if len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf {
		data = data[3:]
	}
	if !utf8.Valid(data) {
		// Best-effort: replace invalid sequences rather than reject UTF-16-ish paste mistakes.
		data = bytes.ToValidUTF8(data, []byte("�"))
	}
	if looksBinary(data) {
		return "", unsupported("content looks binary; use .md/.txt/.docx/.xlsx/.pdf or paste plain text")
	}
	return collapseBlankLines(string(data)), nil
}
