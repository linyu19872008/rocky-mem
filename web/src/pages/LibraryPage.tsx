import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { PageHeader } from "../components/PageHeader";
import { FilterChips } from "../components/FilterChips";
import { TrustBadge } from "../components/Badge";
import { EmptyState } from "../components/EmptyState";
import { Button } from "../components/Button";

type BrowseItem = {
  uid: string;
  kind: string;
  title: string;
  trust?: string;
  subtype?: string;
  preview?: string;
  pinned?: boolean;
  updated_at: number;
};

type BrowseResult = {
  items: BrowseItem[];
  total: number;
};

type KindFilter = "all" | "memory" | "experience";
type TrustFilter = "active" | "proposed" | "all";

function kindsFor(filter: KindFilter): string[] | undefined {
  if (filter === "memory") return ["memory"];
  if (filter === "experience") return ["experience"];
  return undefined;
}

function trustsFor(filter: TrustFilter): string[] {
  if (filter === "proposed") return ["proposed", "contested"];
  if (filter === "all") return ["active", "proposed", "contested"];
  return ["active"];
}

export function LibraryPage() {
  const intl = useIntl();
  const [params, setParams] = useSearchParams();
  const kindParam = params.get("kind");
  const trustParam = params.get("trust");
  const kind: KindFilter =
    kindParam === "memory" || kindParam === "experience" || kindParam === "all" ? kindParam : "all";
  const trust: TrustFilter =
    trustParam === "proposed" || trustParam === "all" || trustParam === "active" ? trustParam : "active";
  const [q, setQ] = useState(params.get("q") || "");
  const [draftQ, setDraftQ] = useState(params.get("q") || "");
  const [items, setItems] = useState<BrowseItem[]>([]);
  const [total, setTotal] = useState(0);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const setKind = (k: KindFilter) => {
    const next = new URLSearchParams(params);
    if (k === "all") next.delete("kind");
    else next.set("kind", k);
    setParams(next, { replace: true });
  };

  const setTrust = (t: TrustFilter) => {
    const next = new URLSearchParams(params);
    if (t === "active") next.delete("trust");
    else next.set("trust", t);
    setParams(next, { replace: true });
  };

  const load = useCallback(async () => {
    setBusy(true);
    setErr("");
    try {
      const out = await api<BrowseResult>("/api/v1/browse", {
        kinds: kindsFor(kind),
        trusts: trustsFor(trust),
        q: q || undefined,
        limit: 50,
      });
      setItems(out.items || []);
      setTotal(out.total || 0);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(
        intl.formatMessage({
          id: `error.${code}`,
          defaultMessage: intl.formatMessage({ id: "error.generic" }),
        }),
      );
      setItems([]);
      setTotal(0);
    } finally {
      setBusy(false);
    }
  }, [intl, kind, trust, q]);

  useEffect(() => {
    void load();
  }, [load]);

  const kindChips = useMemo(
    () => [
      { id: "all" as const, label: intl.formatMessage({ id: "library.kind.all" }) },
      { id: "memory" as const, label: intl.formatMessage({ id: "library.kind.memory" }) },
      { id: "experience" as const, label: intl.formatMessage({ id: "library.kind.experience" }) },
    ],
    [intl],
  );

  const trustChips = useMemo(
    () => [
      { id: "active" as const, label: intl.formatMessage({ id: "library.trust.active" }) },
      { id: "proposed" as const, label: intl.formatMessage({ id: "library.trust.proposed" }) },
      { id: "all" as const, label: intl.formatMessage({ id: "library.trust.all" }) },
    ],
    [intl],
  );

  function onFilterSubmit(e: FormEvent) {
    e.preventDefault();
    const next = new URLSearchParams(params);
    const trimmed = draftQ.trim();
    if (trimmed) next.set("q", trimmed);
    else next.delete("q");
    setParams(next, { replace: true });
    setQ(trimmed);
  }

  return (
    <>
      <PageHeader
        title={<FormattedMessage id="library.title" />}
        actions={
          <Button variant="primary" disabled={busy} onClick={() => void load()}>
            <FormattedMessage id="common.refresh" />
          </Button>
        }
      />
      <p className="meta" style={{ marginTop: 0 }}>
        <FormattedMessage id="library.blurb" />
      </p>

      <FilterChips items={kindChips} value={kind} onChange={setKind} />
      {kind !== "memory" ? <FilterChips items={trustChips} value={trust} onChange={setTrust} /> : null}

      <form className="search-field" onSubmit={onFilterSubmit}>
        <input
          value={draftQ}
          onChange={(e) => setDraftQ(e.target.value)}
          placeholder={intl.formatMessage({ id: "library.filterPlaceholder" })}
        />
        <Button type="submit" variant="primary" disabled={busy}>
          <FormattedMessage id="library.filter" />
        </Button>
      </form>

      {err ? <p className="err">{err}</p> : null}
      <p className="meta">
        <FormattedMessage id="library.count" values={{ shown: items.length, total }} />
      </p>

      {!busy && items.length === 0 ? (
        <EmptyState>
          <FormattedMessage id="library.empty" />{" "}
          <Link to="/ingest">
            <FormattedMessage id="library.emptyIngest" />
          </Link>
        </EmptyState>
      ) : (
        <div className="hit-list">
          {items.map((it) => (
            <article key={it.uid} className="hit-card">
              <header>
                <h3>
                  <Link to={`/entry?uid=${encodeURIComponent(it.uid)}`}>{it.title || it.uid}</Link>
                  {it.pinned ? (
                    <span className="badge active" style={{ marginInlineStart: "0.35rem" }}>
                      <FormattedMessage id="library.pinned" />
                    </span>
                  ) : null}
                </h3>
                <div className="hit-meta">
                  <span className="badge">
                    <FormattedMessage
                      id={it.kind === "memory" ? "library.kind.memory" : "library.kind.experience"}
                    />
                  </span>
                  {it.subtype ? <span className="meta">{it.subtype}</span> : null}
                  {it.trust ? <TrustBadge trust={it.trust} /> : null}
                </div>
              </header>
              {it.preview ? <p>{it.preview}</p> : null}
              <p className="meta mono">{it.uid}</p>
            </article>
          ))}
        </div>
      )}
    </>
  );
}
