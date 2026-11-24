import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { PageHeader } from "../components/PageHeader";
import { DataCard } from "../components/DataCard";
import { Button } from "../components/Button";
import { TrustBadge } from "../components/Badge";

type RecordResult = {
  uid: string;
  trust?: string;
  status?: string;
  review_required?: boolean;
  warnings?: string[];
};

export function RecordPage() {
  const intl = useIntl();
  const nav = useNavigate();
  const [title, setTitle] = useState("");
  const [expType, setExpType] = useState("howto");
  const [symptom, setSymptom] = useState("");
  const [fix, setFix] = useState("");
  const [appliesWhen, setAppliesWhen] = useState("");
  const [invalidWhen, setInvalidWhen] = useState("");
  const [evidence, setEvidence] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [done, setDone] = useState<RecordResult | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    setDone(null);
    try {
      const out = await api<RecordResult>("/api/v1/record", {
        title: title.trim(),
        exp_type: expType,
        symptom: symptom.trim() || undefined,
        fix: fix.trim() || undefined,
        applies_when: appliesWhen.trim() || undefined,
        invalid_when: invalidWhen.trim() || undefined,
        outcome: "unknown",
        confidence: 0.6,
        scope: "global",
        evidence: [{ type: "quote", text: evidence.trim() }],
      });
      setDone(out);
      setTitle("");
      setSymptom("");
      setFix("");
      setAppliesWhen("");
      setInvalidWhen("");
      setEvidence("");
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
      <PageHeader title={<FormattedMessage id="record.title" />} />
      <p className="meta" style={{ marginTop: 0 }}>
        <FormattedMessage id="record.blurb" />
      </p>

      {done ? (
        <DataCard title={<FormattedMessage id="record.done" />}>
          <p>
            <Link to={`/entry?uid=${encodeURIComponent(done.uid)}`}>{done.uid}</Link>
            {done.trust ? (
              <>
                {" "}
                <TrustBadge trust={done.trust} />
              </>
            ) : null}
            {done.review_required ? ` · ${intl.formatMessage({ id: "record.reviewRequired" })}` : ""}
          </p>
          {done.warnings?.length ? (
            <p className="meta">
              <FormattedMessage id="record.warnings" />: {done.warnings.join("; ")}
            </p>
          ) : null}
          <div className="ingest-actions" style={{ border: 0, paddingTop: 0 }}>
            {done.review_required || done.trust === "proposed" || done.trust === "contested" ? (
              <Button variant="primary" onClick={() => nav("/review")}>
                <FormattedMessage id="record.openReview" />
              </Button>
            ) : (
              <Button variant="primary" onClick={() => nav("/library?kind=experience")}>
                <FormattedMessage id="record.openLibrary" />
              </Button>
            )}
            <Button onClick={() => setDone(null)}>
              <FormattedMessage id="record.again" />
            </Button>
          </div>
        </DataCard>
      ) : null}

      {!done ? (
        <DataCard title={<FormattedMessage id="record.form" />}>
          {err ? <p className="err">{err}</p> : null}
          <form onSubmit={(e) => void onSubmit(e)}>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldTitle" />
              </b>
              <input className="field-control" value={title} onChange={(e) => setTitle(e.target.value)} required disabled={busy} />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldType" />
              </b>
              <select className="field-control" value={expType} onChange={(e) => setExpType(e.target.value)} disabled={busy}>
                <option value="howto">howto</option>
                <option value="pitfall">pitfall</option>
                <option value="decision">decision</option>
                <option value="postmortem">postmortem</option>
              </select>
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldSymptom" />
              </b>
              <input className="field-control" value={symptom} onChange={(e) => setSymptom(e.target.value)} disabled={busy} />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldFix" />
              </b>
              <textarea className="field-control" rows={4} value={fix} onChange={(e) => setFix(e.target.value)} disabled={busy} />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldApplies" />
              </b>
              <input className="field-control" value={appliesWhen} onChange={(e) => setAppliesWhen(e.target.value)} disabled={busy} />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldInvalid" />
              </b>
              <input className="field-control" value={invalidWhen} onChange={(e) => setInvalidWhen(e.target.value)} disabled={busy} />
            </label>
            <label className="field field-stack">
              <b>
                <FormattedMessage id="record.fieldEvidence" />
              </b>
              <textarea
                className="field-control"
                rows={4}
                value={evidence}
                onChange={(e) => setEvidence(e.target.value)}
                required
                disabled={busy}
                placeholder={intl.formatMessage({ id: "record.evidenceHint" })}
              />
            </label>
            <div style={{ marginTop: "1rem" }}>
              <Button type="submit" variant="primary" disabled={busy || !title.trim() || !evidence.trim()}>
                <FormattedMessage id="record.submit" />
              </Button>
            </div>
          </form>
        </DataCard>
      ) : null}
    </>
  );
}
