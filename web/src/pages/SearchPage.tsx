import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { Button } from "../components/Button";
import { PageHeader } from "../components/PageHeader";
import { DataCard } from "../components/DataCard";
import { TrustBadge } from "../components/Badge";

type HitSummary = {
  uid: string;
  kind: string;
  title: string;
  trust?: string;
  applies_when?: string;
  invalid_when?: string;
  preview?: string;
};

type PackResult = {
  pack?: string;
  uids?: string[];
  hits?: HitSummary[];
};

export function SearchPage() {
  const intl = useIntl();
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<HitSummary[]>([]);
  const [pack, setPack] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [searched, setSearched] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    try {
      const data = await api<PackResult>("/api/v1/search", {
        query: q,
        include_proposed: true,
        budget_tokens: 1200,
      });
      const cards =
        data.hits && data.hits.length
          ? data.hits
          : (data.uids || []).map((uid) => ({ uid, kind: "unknown", title: uid }));
      setHits(cards);
      setPack(data.pack || "");
      setSearched(true);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
      setHits([]);
      setPack("");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeader title={<FormattedMessage id="search.title" />} />
      <form className="search-field" onSubmit={onSubmit}>
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={intl.formatMessage({ id: "search.placeholder" })}
          autoFocus
        />
        <Button type="submit" variant="primary" disabled={busy}>
          <FormattedMessage id="search.submit" />
        </Button>
      </form>
      {err ? <p className="err">{err}</p> : null}
      {!searched && !err ? (
        <p className="meta">
          {intl.formatMessage({ id: "search.empty" })}{" "}
          <Link to="/library">
            <FormattedMessage id="search.browseLibrary" />
          </Link>
        </p>
      ) : null}
      {searched ? (
        <p className="meta">
          <FormattedMessage id="search.hits" values={{ count: hits.length }} />
        </p>
      ) : null}
      {hits.length > 0 ? (
        <div className="hit-list">
          {hits.map((h) => (
            <article key={h.uid} className="hit-card">
              <header>
                <h3>
                  <Link to={`/entry?uid=${encodeURIComponent(h.uid)}`}>{h.title || h.uid}</Link>
                </h3>
                <div className="hit-meta">
                  <span className="meta">{h.kind}</span>
                  {h.trust ? <TrustBadge trust={h.trust} /> : null}
                </div>
              </header>
              {h.applies_when ? (
                <p>
                  <b>
                    <FormattedMessage id="search.appliesWhen" />
                  </b>
                  : {h.applies_when}
                </p>
              ) : null}
              {h.invalid_when ? (
                <p>
                  <b>
                    <FormattedMessage id="search.invalidWhen" />
                  </b>
                  : {h.invalid_when}
                </p>
              ) : null}
              {h.preview ? <p className="meta">{h.preview}</p> : null}
              <p className="meta mono">{h.uid}</p>
            </article>
          ))}
        </div>
      ) : null}
      {pack ? (
        <DataCard title={<FormattedMessage id="search.pack" />}>
          <pre className="mono" style={{ maxHeight: "28rem" }}>
            {pack}
          </pre>
        </DataCard>
      ) : null}
    </>
  );
}
