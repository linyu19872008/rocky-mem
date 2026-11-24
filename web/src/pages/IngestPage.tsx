import { useCallback, useState, type DragEvent, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { PageHeader } from "../components/PageHeader";
import { DataCard } from "../components/DataCard";
import { Button } from "../components/Button";

type Preview = {
  filename?: string;
  content_hash: string;
  original: string;
  redacted: string;
  hits?: Record<string, number>;
  sensitivity_level: number;
  rejected: boolean;
  reject_why?: string;
  rule_pack: string;
};

type ConfirmResult = {
  uid: string;
  trust?: string;
  status?: string;
  review_required?: boolean;
};

export function IngestPage() {
  const intl = useIntl();
  const [preview, setPreview] = useState<Preview | null>(null);
  const [title, setTitle] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState<ConfirmResult | null>(null);

  const runPreview = useCallback(
    async (text: string, filename: string) => {
      setBusy(true);
      setErr("");
      setDone(null);
      try {
        const out = await api<Preview>("/api/v1/ingest/preview", { text, filename });
        setPreview(out);
        if (!title && filename) {
          setTitle(`Ingest: ${filename}`);
        }
      } catch (ex) {
        const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
        setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
        setPreview(null);
      } finally {
        setBusy(false);
      }
    },
    [intl, title],
  );

  async function onDrop(e: DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files?.[0];
    if (!file) return;
    const text = await file.text();
    await runPreview(text, file.name);
  }

  async function onPasteSubmit(e: FormEvent) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget as HTMLFormElement);
    const text = String(fd.get("text") || "");
    await runPreview(text, "paste.txt");
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
      });
      setDone(out);
      setPreview(null);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
    } finally {
      setBusy(false);
    }
  }

  const hitEntries = preview?.hits ? Object.entries(preview.hits) : [];

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
          <div
            className="drop-zone"
            onDragOver={(e) => e.preventDefault()}
            onDrop={(ev) => void onDrop(ev)}
          >
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
        <>
          <DataCard title={<FormattedMessage id="ingest.previewTitle" />}>
            <p className="meta">
              {preview.filename || "—"} · L{preview.sensitivity_level} · {preview.rule_pack} ·{" "}
              {preview.content_hash.slice(0, 12)}…
            </p>
            {preview.rejected ? (
              <p className="err">
                <FormattedMessage id="ingest.rejected" /> {preview.reject_why}
              </p>
            ) : null}
            {hitEntries.length ? (
              <p className="meta">
                <FormattedMessage id="ingest.hits" />:{" "}
                {hitEntries.map(([k, n]) => `${k}×${n}`).join(", ")}
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
            <div className="compare ingest-compare">
              <div>
                <h4>
                  <FormattedMessage id="ingest.original" />
                </h4>
                <pre>{preview.original}</pre>
              </div>
              <div>
                <h4>
                  <FormattedMessage id="ingest.redacted" />
                </h4>
                <pre>{preview.redacted}</pre>
              </div>
            </div>
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
        </>
      ) : null}
    </>
  );
}
