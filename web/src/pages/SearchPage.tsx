import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { Button } from "../components/Button";
import { PageHeader } from "../components/PageHeader";
import { DataCard } from "../components/DataCard";

type PackResult = {
  pack?: string;
  uids?: string[];
};

export function SearchPage() {
  const intl = useIntl();
  const [q, setQ] = useState("");
  const [uids, setUids] = useState<string[]>([]);
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
      setUids(data.uids || []);
      setPack(data.pack || "");
      setSearched(true);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
      setUids([]);
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
      {!searched && !err ? <p className="meta">{intl.formatMessage({ id: "search.empty" })}</p> : null}
      {searched ? (
        <p className="meta">
          <FormattedMessage id="search.hits" values={{ count: uids.length }} />
        </p>
      ) : null}
      {uids.length > 0 ? (
        <ul className="hits">
          {uids.map((id) => (
            <li key={id}>
              <Link to={`/entry?uid=${encodeURIComponent(id)}`}>{id}</Link>
            </li>
          ))}
        </ul>
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
