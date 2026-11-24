package extract

import (
	"bytes"
	"path/filepath"
	"strings"
)

// Format is a supported ingest content kind after detection.
type Format string

const (
	FormatPlain Format = "plain"
	FormatDOCX  Format = "docx"
	FormatXLSX  Format = "xlsx"
	FormatPDF   Format = "pdf"
)

// Detect returns the ingest format from filename extension and magic bytes.
func Detect(filename string, data []byte) (Format, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if looksPDF(data) || ext == ".pdf" {
		if !looksPDF(data) && len(data) > 0 {
			return "", unsupported("pdf extension but content is not a PDF")
		}
		return FormatPDF, nil
	}
	if looksZIP(data) {
		switch ext {
		case ".docx":
			return FormatDOCX, nil
		case ".xlsx":
			return FormatXLSX, nil
		case ".doc", ".ppt", ".pptx", ".odt", ".ods":
			return "", unsupported("office format " + ext + " is not supported; save as .docx/.xlsx/.pdf/.md/.txt")
		default:
			if ext == ".zip" {
				return "", unsupported("raw .zip is not supported")
			}
			// ZIP without known office ext — still try docx/xlsx by sniffing later
			if bytes.Contains(data[:min(4096, len(data))], []byte("word/")) {
				return FormatDOCX, nil
			}
			if bytes.Contains(data[:min(4096, len(data))], []byte("xl/")) {
				return FormatXLSX, nil
			}
			return "", unsupported("zip archive is not a supported document type")
		}
	}
	switch ext {
	case ".docx":
		return "", unsupported("docx extension but content is not a ZIP/OOXML package")
	case ".xlsx":
		return "", unsupported("xlsx extension but content is not a ZIP/OOXML package")
	case ".doc", ".ppt", ".pptx", ".odt", ".rtf":
		return "", unsupported("format " + ext + " is not supported; export to .docx/.pdf/.md/.txt")
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bin":
		return "", unsupported("binary/image formats are not supported (no OCR)")
	}
	if looksBinary(data) {
		return "", unsupported("content looks binary; use .md/.txt/.docx/.xlsx/.pdf or paste plain text")
	}
	return FormatPlain, nil
}

func looksPDF(data []byte) bool {
	if len(data) < 5 {
		return false
	}
	return bytes.HasPrefix(data, []byte("%PDF-"))
}

func looksZIP(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && (data[2] == 3 || data[2] == 5 || data[2] == 7)
}

func looksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	n := min(8192, len(data))
	nul := 0
	ctrl := 0
	for i := 0; i < n; i++ {
		b := data[i]
		if b == 0 {
			nul++
		}
		if b < 0x09 || (b > 0x0d && b < 0x20) {
			ctrl++
		}
	}
	if nul > 0 {
		return true
	}
	return float64(ctrl)/float64(n) > 0.30
}

func unsupported(msg string) error {
	return &Error{Code: "unsupported", Message: msg}
}

// Error is a typed extract failure.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }
