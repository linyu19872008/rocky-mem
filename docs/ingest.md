# File ingest (two-phase)

Pipeline:

```
bytes → detect (ext + magic) → extract plain text → redact → chunk
      → preview → confirm → source + chunks (+ optional proposed experience)
```

Confirm has two modes:

| Mode | What is stored | Default |
|---|---|---|
| `corpus` | `source` + its `chunk` rows + retrieval docs. **Background material, not a claim** | ✅ |
| `distill` | the above, plus one `proposed` experience for review | opt-in |

Corpus is the default because a document is not a lesson. Distillation keeps the
old behaviour (title from the filename, first line as symptom, body as the fix)
and is meant for short notes the human actually vouches for.

Re-ingesting the same text is idempotent: the response is `unchanged` and reuses
the existing source. Re-ingesting the **same filename with different content**
archives the previous source and removes its chunks from retrieval, so two
generations never compete.

## Supported

| Format | Notes |
|---|---|
| `.md` `.txt` `.json` `.yaml` `.yml` `.csv` `.log` | UTF-8 plain text |
| `.docx` | OOXML: paragraphs + tables as TSV rows (`tbl/tr/tc`) |
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
{ "redacted": "…", "filename": "a.docx", "format": "docx", "title": "…",
  "mode": "corpus" }
```

Response (corpus):

```json
{ "mode": "corpus", "source_uid": "s_…", "source_title": "a",
  "chunk_count": 7, "archived_previous": 1, "unchanged": false }
```

`distill` adds `uid` / `status` / `trust` / `review_required` for the proposed
experience. Read passages back with `mem_get(uid: "c_…")`, and list a document's
chunks with `mem_get(uid: "s_…")`.

Preview never returns original binary; `original` is **extracted text** only.

Extra preview fields for the UI:

| Field | Meaning |
|---|---|
| `unchanged` | No redaction hits — show a single extracted pane (no fake dual diff) |
| `spans` | Byte offsets of each replacement on `original` |
| `segments` | Inline diff pieces: `eq` / `del` / `ins` (+ optional `rule`) |

Web: zero-hit → single pane; hits → one colored inline diff (`del` strike / `ins` green).
Preview chrome shows filename + status chip; hash/rulepack live under “Technical details”.
Tab-separated extract renders as a table when present.
