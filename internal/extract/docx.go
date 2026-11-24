package extract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

func extractDOCX(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", unsupported("invalid docx/zip: " + err.Error())
	}
	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", unsupported("docx missing word/document.xml")
	}
	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, 20<<20))
	if err != nil {
		return "", err
	}
	return docxXMLToText(raw), nil
}

func docxXMLToText(raw []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tab":
				b.WriteByte('\t')
			case "br", "cr":
				b.WriteByte('\n')
			case "p":
				if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
					b.WriteByte('\n')
				}
			}
		case xml.CharData:
			s := string(t)
			if strings.TrimSpace(s) == "" && strings.ContainsAny(s, "\n\r") {
				continue
			}
			b.WriteString(s)
		}
	}
	return collapseBlankLines(b.String())
}

func collapseBlankLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}
