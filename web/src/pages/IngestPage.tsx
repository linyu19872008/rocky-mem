import { useCallback, useState, type DragEvent, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { PageHeader } from "../components/PageHeader";
import { DataCard } from "../components/DataCard";
import { Button } from "../components/Button";
import { RedactDiff, type DiffSegment } from "../components/RedactDiff";

type Preview = {
  filename?: string;
  format?: string;
  content_hash: string;
  char_count?: number;
  chunk_count?: number;
  original: string;
  redacted: string;
  segments?: DiffSegment[];
  spans?: Array<{ rule: string; start: number; end: number; original: string; replacement: string }>;
  hits?: Record<string, number>;
  sensitivity_level: number;
  rejected: boolean;
  reject_why?: string;
  rule_pack: string;
  notice?: string;
  unchanged?: boolean;
};

type ConfirmResult = {
  uid: string;
  trust?: string;
  status?: string;
  review_required?: boolean;
};

async function fileToBase64(file: File): Promise<string> {
  const buf = await file.arrayBuffer();
  const bytes = new Uint8Array(buf);
  let binary = "";
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}

export function IngestPage() {
  const intl = useIntl();
  const [preview, setPreview] = useState<Preview | null>(null);
  const [title, setTitle] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState<ConfirmResult | null>(null);

  const runPreviewBytes = useCallback(
    async (contentBase64: string, filename: string, contentType?: string) => {
      setBusy(true);
      setErr("");
      setDone(null);
      try {
        const out = await api<Preview>("/api/v1/ingest/preview", {
          content_base64: contentBase64,
          filename,
          content_type: contentType,
        });
        setPreview(out);
        setTitle((prev) => prev || (filename ? `Ingest: ${filename}` : ""));
      } catch (ex) {
        const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
        const msg =
          ex instanceof ApiRequestError && ex.message
            ? ex.message
            : intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) });
        setErr(msg);
        setPreview(null);
      } finally {
        setBusy(false);
      }
    },
    [intl],
  );

  const runPreviewText = useCallback(
    async (text: string, filename: string) => {
      setBusy(true);
      setErr("");
      setDone(null);
      try {
        const out = await api<Preview>("/api/v1/ingest/preview", { text, filename });
        setPreview(out);
        setTitle((prev) => prev || (filename ? `Ingest: ${filename}` : ""));
      } catch (ex) {
        const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
        const msg =
          ex instanceof ApiRequestError && ex.message
            ? ex.message
            : intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) });
        setErr(msg);
        setPreview(null);
      } finally {
        setBusy(false);
      }
    },
    [intl],
  );

  async function onDrop(e: DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files?.[0];
    if (!file) return;
    if (file.size > 15 * 1024 * 1024) {
      setErr(intl.formatMessage({ id: "ingest.tooLarge" }));
      return;
    }
    const b64 = await fileToBase64(file);
    await runPreviewBytes(b64, file.name, file.type);
  }

  async function onPasteSubmit(e: FormEvent) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget as HTMLFormElement);
    const text = String(fd.get("text") || "");
    await runPreviewText(text, "paste.txt");
  }

  async function onConfirm() {
    if (!preview || preview.rejected) return;
    setBusy(true);
    setErr("");
    try {
      const out = await api<ConfirmResult>("/api/v1/ingest/confirm", {
        redacted: preview.redacted,
        filename: preview.filename,
        title: title.trim() || undefined,
        format: preview.format,
      });
      setDone(out);
      setPreview(null);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      const msg =
        ex instanceof ApiRequestError && ex.message
          ? ex.message
          : intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) });
      setErr(msg);
    } finally {
      setBusy(false);
    }
  }

  const hitEntries = preview?.hits ? Object.entries(preview.hits) : [];
  const unchanged = Boolean(preview?.unchanged) || hitEntries.length === 0;
  const segments = preview?.segments?.length
    ? preview.segments
    : preview
      ? ([{ kind: "eq", text: preview.original }] as DiffSegment[])
      : [];

  return (
    <>
      <PageHeader title={<FormattedMessage id="ingest.title" />} />
      {err ? <p className="err">{err}</p> : null}
      {done ? (
        <DataCard title={<FormattedMessage id="ingest.done" />}>
          <p>
            <Link to={`/entry?uid=${encodeURIComponent(done.uid)}`}>{done.uid}</Link>
            {done.trust ? ` · ${done.trust}` : ""}
            {done.review_required ? ` · ${intl.formatMessage({ id: "ingest.reviewRequired" })}` : ""}
          </p>
          <Button variant="primary" onClick={() => setDone(null)}>
            <FormattedMessage id="ingest.again" />
          </Button>
        </DataCard>
      ) : null}

      {!preview && !done ? (
        <>
          <div className="drop-zone" onDragOver={(e) => e.preventDefault()} onDrop={(ev) => void onDrop(ev)}>
            <div style={{ fontWeight: 650, marginBottom: "0.5rem" }}>
              <FormattedMessage id="ingest.drop" />
            </div>
            <p className="meta" style={{ margin: 0 }}>
              <FormattedMessage id="ingest.hint" />
            </p>
          </div>
          <DataCard title={<FormattedMessage id="ingest.paste" />}>
            <form onSubmit={(e) => void onPasteSubmit(e)}>
              <textarea name="text" rows={8} style={{ width: "100%" }} disabled={busy} />
              <div style={{ marginTop: "0.75rem" }}>
                <Button type="submit" variant="primary" disabled={busy}>
                  <FormattedMessage id="ingest.preview" />
                </Button>
              </div>
            </form>
          </DataCard>
        </>
      ) : null}

      {preview ? (
        <DataCard title={<FormattedMessage id="ingest.previewTitle" />}>
          <p className="meta">
            {preview.filename || "—"} · {preview.format || "?"} · L{preview.sensitivity_level} ·{" "}
            {preview.rule_pack} · {preview.char_count ?? "?"} chars · {preview.chunk_count ?? "?"} chunks ·{" "}
            {preview.content_hash.slice(0, 12)}…
          </p>
          {preview.notice ? <p className="meta">{preview.notice}</p> : null}
          {preview.rejected ? (
            <p className="err">
              <FormattedMessage id="ingest.rejected" /> {preview.reject_why}
            </p>
          ) : null}
          {hitEntries.length ? (
            <p className="meta">
              <FormattedMessage id="ingest.hits" />: {hitEntries.map(([k, n]) => `${k}×${n}`).join(", ")}
            </p>
          ) : (
            <p className="meta">
              <FormattedMessage id="ingest.noHits" />
            </p>
          )}
          <label className="field">
            <b>
              <FormattedMessage id="ingest.titleField" />
            </b>
            <input value={title} onChange={(e) => setTitle(e.target.value)} disabled={busy} />
          </label>

          {unchanged ? (
            <div className="ingest-single">
              <h4>
                <FormattedMessage id="ingest.extracted" />
              </h4>
              <pre className="mono ingest-body">{preview.original}</pre>
            </div>
          ) : (
            <div className="ingest-diff">
              <h4>
                <FormattedMessage id="ingest.diff" />
              </h4>
              <RedactDiff segments={segments} />
            </div>
          )}

          <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.75rem" }}>
            <Button variant="primary" disabled={busy || preview.rejected} onClick={() => void onConfirm()}>
              <FormattedMessage id="ingest.confirm" />
            </Button>
            <Button
              disabled={busy}
              onClick={() => {
                setPreview(null);
                setErr("");
              }}
            >
              <FormattedMessage id="ingest.cancel" />
            </Button>
          </div>
        </DataCard>
      ) : null}
    </>
  );
}
