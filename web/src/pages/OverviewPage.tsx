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

export function OverviewPage() {
  const intl = useIntl();
  const nav = useNavigate();
  const [pending, setPending] = useState<ReviewItem[]>([]);
  const [doctorOk, setDoctorOk] = useState<boolean | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const [rows, doc] = await Promise.all([
          api<ReviewItem[]>("/api/v1/review/list", { trusts: ["proposed", "contested"], limit: 8 }),
          apiGet<Record<string, unknown>>("/api/v1/doctor"),
        ]);
        setPending(rows || []);
        setDoctorOk(doc.search_doc_ok !== false);
      } catch (ex) {
        const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
        setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
      }
    })();
  }, [intl]);

  const proposed = pending.filter((r) => r.trust === "proposed").length;
  const contested = pending.filter((r) => r.trust === "contested").length;

  return (
    <>
      <PageHeader
        title={<FormattedMessage id="overview.title" />}
        actions={
          <Button variant="primary" onClick={() => nav("/review")}>
            <FormattedMessage id="overview.openReview" />
          </Button>
        }
      />
      {err ? <p className="err">{err}</p> : null}
      <StatStrip
        items={[
          {
            label: intl.formatMessage({ id: "overview.stat.memories" }),
            value: intl.formatMessage({ id: "overview.stat.unknown" }),
          },
          {
            label: intl.formatMessage({ id: "overview.stat.experiences" }),
            value: intl.formatMessage({ id: "overview.stat.unknown" }),
          },
          {
            label: intl.formatMessage({ id: "overview.stat.pending" }),
            value: pending.length || proposed + contested,
          },
          {
            label: intl.formatMessage({ id: "overview.stat.doctor" }),
            value:
              doctorOk == null
                ? intl.formatMessage({ id: "overview.stat.unknown" })
                : doctorOk
                  ? intl.formatMessage({ id: "overview.stat.ok" })
                  : intl.formatMessage({ id: "overview.stat.bad" }),
          },
        ]}
      />
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
