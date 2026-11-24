package extract

import (
	"bytes"
	"io"
	"unicode"

	"github.com/ledongthuc/pdf"
)

func extractPDF(data []byte) (string, error) {
	if len(data) == 0 {
		return "", unsupported("empty pdf")
	}
	var (
		text string
		err  error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = unsupported("pdf parse failed (malformed or encrypted)")
			}
		}()
		r, e := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
		if e != nil {
			err = unsupported("pdf open failed: " + e.Error())
			return
		}
		plain, e := r.GetPlainText()
		if e != nil {
			err = unsupported("pdf text extract failed: " + e.Error())
			return
		}
		var buf bytes.Buffer
		if _, e := io.Copy(&buf, io.LimitReader(plain, 8<<20)); e != nil {
			err = e
			return
		}
		text = buf.String()
	}()
	if err != nil {
		return "", err
	}
	text = collapseBlankLines(text)
	if !hasSubstantialText(text) {
		return "", unsupported("scanned PDF or no text layer (OCR not supported); export text or use a text-based PDF")
	}
	return text, nil
}

func hasSubstantialText(s string) bool {
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.Is(unicode.Han, r) {
			letters++
			if letters >= 20 {
				return true
			}
		}
	}
	return letters >= 20
}
