import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { PageHeader } from "../components/PageHeader";
import { DataCard } from "../components/DataCard";
import { Button } from "../components/Button";

type RememberResult = {
  uid: string;
  lineage?: string;
  status?: string;
  rev?: number;
  review_required?: boolean;
};

export function RememberPage() {
  const intl = useIntl();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [memKey, setMemKey] = useState("");
  const [memType, setMemType] = useState("pref");
  const [pinned, setPinned] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [done, setDone] = useState<RememberResult | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    setDone(null);
    try {
      const out = await api<RememberResult>("/api/v1/remember", {
        title: title.trim(),
        body: body.trim(),
        mem_key: memKey.trim() || undefined,
        mem_type: memType || undefined,
        pinned,
        scope: "global",
      });
      setDone(out);
      setTitle("");
      setBody("");
      setMemKey("");
      setPinned(false);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(
        ex instanceof ApiRequestError && ex.message
          ? ex.message
          : intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeader title={<FormattedMessage id="remember.title" />} />
      <p className="meta" style={{ marginTop: 0 }}>
        <FormattedMessage id="remember.blurb" />
      </p>

      {done ? (
        <DataCard title={<FormattedMessage id="remember.done" />}>
          <p>
            <Link to={`/entry?uid=${encodeURIComponent(done.uid)}`}>{done.uid}</Link>
            {done.status ? ` · ${done.status}` : ""}
            {done.rev != null ? ` · rev ${done.rev}` : ""}
          </p>
          <p className="meta">
            <Link to="/library?kind=memory">
              <FormattedMessage id="remember.viewLibrary" />
            </Link>
          </p>
          <Button variant="primary" onClick={() => setDone(null)}>
            <FormattedMessage id="remember.again" />
          </Button>
        </DataCard>
      ) : null}

      {!done ? (
        <DataCard title={<FormattedMessage id="remember.form" />}>
          {err ? <p className="err">{err}</p> : null}
          <form onSubmit={(e) => void onSubmit(e)}>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="remember.fieldTitle" />
              </b>
              <input
                className="field-control"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
                disabled={busy}
              />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="remember.fieldBody" />
              </b>
              <textarea
                className="field-control"
                rows={6}
                value={body}
                onChange={(e) => setBody(e.target.value)}
                required
                disabled={busy}
              />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="remember.fieldKey" />
              </b>
              <input
                className="field-control"
                value={memKey}
                onChange={(e) => setMemKey(e.target.value)}
                placeholder={intl.formatMessage({ id: "remember.keyHint" })}
                disabled={busy}
                spellCheck={false}
              />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="remember.fieldType" />
              </b>
              <select
                className="field-control"
                value={memType}
                onChange={(e) => setMemType(e.target.value)}
                disabled={busy}
              >
                <option value="pref">pref</option>
                <option value="fact">fact</option>
                <option value="constraint">constraint</option>
                <option value="glossary">glossary</option>
              </select>
            </label>
            <label className="field" style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginTop: "0.75rem" }}>
              <input type="checkbox" checked={pinned} onChange={(e) => setPinned(e.target.checked)} disabled={busy} />
              <span>
                <FormattedMessage id="remember.pinned" />
              </span>
            </label>
            <div style={{ marginTop: "1rem" }}>
              <Button type="submit" variant="primary" disabled={busy || !title.trim() || !body.trim()}>
                <FormattedMessage id="remember.submit" />
              </Button>
            </div>
          </form>
        </DataCard>
      ) : null}
    </>
  );
}
