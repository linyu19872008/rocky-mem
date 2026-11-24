package extract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func extractXLSX(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", unsupported("invalid xlsx/zip: " + err.Error())
	}
	shared, err := readSharedStrings(zr)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	sheetN := 0
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "xl/worksheets/sheet") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		sheetN++
		rc, err := f.Open()
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(rc, 20<<20))
		_ = rc.Close()
		if err != nil {
			continue
		}
		b.WriteString(fmt.Sprintf("## Sheet %d\n", sheetN))
		b.WriteString(sheetXMLToText(raw, shared))
		b.WriteByte('\n')
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", unsupported("xlsx contained no readable cells")
	}
	return out, nil
}

func readSharedStrings(zr *zip.Reader) ([]string, error) {
	var f *zip.File
	for _, zf := range zr.File {
		if zf.Name == "xl/sharedStrings.xml" {
			f = zf
			break
		}
	}
	if f == nil {
		return nil, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, 20<<20))
	if err != nil {
		return nil, err
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var out []string
	var cur strings.Builder
	inT := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				cur.Reset()
			}
			if t.Name.Local == "t" {
				inT = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			}
			if t.Name.Local == "si" {
				out = append(out, cur.String())
			}
		case xml.CharData:
			if inT {
				cur.Write(t)
			}
		}
	}
	return out, nil
}

func sheetXMLToText(raw []byte, shared []string) string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var b strings.Builder
	var row []string
	var cellType string
	var cellVal strings.Builder
	inV := false
	flushCell := func() {
		v := strings.TrimSpace(cellVal.String())
		cellVal.Reset()
		if cellType == "s" {
			if i, err := strconv.Atoi(v); err == nil && i >= 0 && i < len(shared) {
				v = shared[i]
			}
		}
		row = append(row, v)
		cellType = ""
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				row = row[:0]
			case "c":
				cellType = ""
				for _, a := range t.Attr {
					if a.Name.Local == "t" {
						cellType = a.Value
					}
				}
			case "v", "t":
				inV = true
				cellVal.Reset()
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v", "t":
				inV = false
				flushCell()
			case "c":
				// empty cell may have no v
			case "row":
				line := strings.TrimRight(strings.Join(row, "\t"), "\t")
				if strings.TrimSpace(line) != "" {
					b.WriteString(line)
					b.WriteByte('\n')
				}
			}
		case xml.CharData:
			if inV {
				cellVal.Write(t)
			}
		}
	}
	return b.String()
}
