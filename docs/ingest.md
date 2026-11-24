# File ingest (two-phase)

Pipeline:

```
bytes → detect (ext + magic) → extract plain text → chunk → redact preview → confirm
```

## Supported

| Format | Notes |
|---|---|
| `.md` `.txt` `.json` `.yaml` `.yml` `.csv` `.log` | UTF-8 plain text |
| `.docx` | OOXML: `word/document.xml` paragraphs |
| `.xlsx` | Shared strings + sheet cells (tab-separated rows) |
| `.pdf` | **Text layer only** (pure Go). Scanned/image PDFs fail clearly |

Max upload: **15 MiB**. Extracted text capped ~200k runes.

## Not supported (explicit errors)

Old `.doc` / `.ppt` / `.odt`, raw `.zip`, images, encrypted PDF, scanned PDF (no OCR).

## API

```http
POST /api/v1/ingest/preview
{ "content_base64": "…", "filename": "a.docx" }
# or paste:
{ "text": "…", "filename": "paste.txt" }

POST /api/v1/ingest/confirm
{ "redacted": "…", "filename": "a.docx", "format": "docx", "title": "…" }
```

Preview never returns original binary; `original` is **extracted text** only.
