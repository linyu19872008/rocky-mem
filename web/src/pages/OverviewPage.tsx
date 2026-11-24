import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, apiGet, ApiRequestError } from "../lib/api";
import { PageHeader } from "../components/PageHeader";
import { StatStrip } from "../components/StatStrip";
import { DataCard } from "../components/DataCard";
import { EmptyState } from "../components/EmptyState";
import { Button } from "../components/Button";
import { TrustBadge } from "../components/Badge";

type ReviewItem = {
  uid: string;
  title: string;
  trust: string;
};

type BrowseItem = {
  uid: string;
  kind: string;
  title: string;
  trust?: string;
  preview?: string;
};

type Stats = {
  active_memories: number;
  experiences: number;
  pending_review: number;
  proposed: number;
  contested: number;
  doctor_ok: boolean;
  schema_version?: string;
};

export function OverviewPage() {
  const intl = useIntl();
  const nav = useNavigate();
  const [pending, setPending] = useState<ReviewItem[]>([]);
  const [recent, setRecent] = useState<BrowseItem[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const [rows, st, lib] = await Promise.all([
          api<ReviewItem[]>("/api/v1/review/list", { trusts: ["proposed", "contested"], limit: 8 }),
          apiGet<Stats>("/api/v1/stats"),
          api<{ items: BrowseItem[] }>("/api/v1/browse", {
            kinds: ["memory", "experience"],
            trusts: ["active"],
            limit: 8,
          }),
        ]);
        setPending(rows || []);
        setStats(st);
        setRecent(lib.items || []);
      } catch (ex) {
        const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
        setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
      }
    })();
  }, [intl]);

  const unknown = intl.formatMessage({ id: "overview.stat.unknown" });

  return (
    <>
      <PageHeader
        title={<FormattedMessage id="overview.title" />}
        actions={
          <>
            <Button onClick={() => nav("/library")}>
              <FormattedMessage id="overview.openLibrary" />
            </Button>
            <Button variant="primary" onClick={() => nav("/review")}>
              <FormattedMessage id="overview.openReview" />
            </Button>
          </>
        }
      />
      {err ? <p className="err">{err}</p> : null}
      <StatStrip
        items={[
          {
            label: intl.formatMessage({ id: "overview.stat.memories" }),
            value: stats == null ? unknown : stats.active_memories,
            onClick: () => nav("/library?kind=memory"),
          },
          {
            label: intl.formatMessage({ id: "overview.stat.experiences" }),
            value: stats == null ? unknown : stats.experiences,
            onClick: () => nav("/library?kind=experience"),
          },
          {
            label: intl.formatMessage({ id: "overview.stat.pending" }),
            value: stats == null ? unknown : stats.pending_review,
            onClick: () => nav("/library?kind=experience&trust=proposed"),
          },
          {
            label: intl.formatMessage({ id: "overview.stat.doctor" }),
            value:
              stats == null
                ? unknown
                : stats.doctor_ok
                  ? intl.formatMessage({ id: "overview.stat.ok" })
                  : intl.formatMessage({ id: "overview.stat.bad" }),
          },
        ]}
      />

      <DataCard title={<FormattedMessage id="overview.recentActive" />}>
        {!recent.length ? (
          <EmptyState>
            <FormattedMessage id="overview.recentActiveEmpty" />{" "}
            <Link to="/ingest">
              <FormattedMessage id="library.emptyIngest" />
            </Link>
          </EmptyState>
        ) : (
          recent.map((r) => (
            <div key={r.uid} className="row" style={{ marginBlockEnd: "0.45rem" }}>
              <div>
                <h3>
                  <Link to={`/entry?uid=${encodeURIComponent(r.uid)}`}>{r.title}</Link>{" "}
                  <span className="badge">
                    <FormattedMessage
                      id={r.kind === "memory" ? "library.kind.memory" : "library.kind.experience"}
                    />
                  </span>{" "}
                  {r.trust ? <TrustBadge trust={r.trust} /> : null}
                </h3>
                {r.preview ? <p>{r.preview}</p> : <p className="meta mono">{r.uid}</p>}
              </div>
            </div>
          ))
        )}
      </DataCard>

      <DataCard title={<FormattedMessage id="overview.recent" />}>
        {!pending.length ? (
          <EmptyState>
            <FormattedMessage id="overview.empty" />
          </EmptyState>
        ) : (
          pending.map((r) => (
            <div key={r.uid} className={`row ${r.trust}`} style={{ marginBlockEnd: "0.45rem" }}>
              <div>
                <h3>
                  <Link to={`/entry?uid=${encodeURIComponent(r.uid)}`}>{r.title}</Link>{" "}
                  <TrustBadge trust={r.trust} />
                </h3>
                <p>{r.uid}</p>
              </div>
            </div>
          ))
        )}
      </DataCard>
    </>
  );
}
