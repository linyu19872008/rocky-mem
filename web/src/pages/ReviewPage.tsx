import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { Button } from "../components/Button";
import { TrustBadge } from "../components/Badge";
import { Toast } from "../components/Toast";
import { PageHeader } from "../components/PageHeader";
import { StatStrip } from "../components/StatStrip";
import { FilterChips } from "../components/FilterChips";

type ReviewItem = {
  uid: string;
  title: string;
  trust: string;
  exp_type?: string;
  confidence?: number;
  symptom?: string;
  conflict_group?: string;
};

type Filter = "all" | "proposed" | "contested";

export function ReviewPage() {
  const intl = useIntl();
  const nav = useNavigate();
  const [rows, setRows] = useState<ReviewItem[]>([]);
  const [filter, setFilter] = useState<Filter>("all");
  const [status, setStatus] = useState(intl.formatMessage({ id: "common.loading" }));
  const [err, setErr] = useState("");
  const [toast, setToast] = useState("");

  const refresh = useCallback(async () => {
    setErr("");
    setStatus(intl.formatMessage({ id: "common.loading" }));
    try {
      const data = await api<ReviewItem[]>("/api/v1/review/list", {
        trusts: ["proposed", "contested"],
        limit: 50,
      });
      setRows(data || []);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
      setRows([]);
    }
  }, [intl]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const proposedCount = useMemo(() => rows.filter((r) => r.trust === "proposed").length, [rows]);
  const contestedCount = useMemo(() => rows.filter((r) => r.trust === "contested").length, [rows]);

  const visible = useMemo(
    () => rows.filter((r) => filter === "all" || r.trust === filter),
    [rows, filter],
  );

  useEffect(() => {
    if (!visible.length) {
      setStatus(
        rows.length
          ? intl.formatMessage({ id: "review.queue.filterEmpty" })
          : intl.formatMessage({ id: "review.queue.empty" }),
      );
    } else {
      setStatus(intl.formatMessage({ id: "review.queue.count" }, { count: visible.length }));
    }
  }, [visible, rows.length, intl]);

  const approve = async (uid: string) => {
    await api("/api/v1/review/approve", { uid });
    setToast(intl.formatMessage({ id: "review.action.approve" }));
    await refresh();
  };
  const reject = async (uid: string) => {
    await api("/api/v1/review/reject", { uid, reason: "web-reject" });
    setToast(intl.formatMessage({ id: "review.action.reject" }));
    await refresh();
  };

  const groups = useMemo(() => {
    const map = new Map<string, ReviewItem[]>();
    for (const r of visible) {
      const key =
        r.conflict_group || (r.trust === "contested" ? "_contested_ungrouped" : "_proposed");
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(r);
    }
    return map;
  }, [visible]);

  return (
    <>
      <PageHeader
        title={<FormattedMessage id="review.title" />}
        actions={
          <Button variant="primary" onClick={() => void refresh()}>
            <FormattedMessage id="review.refresh" />
          </Button>
        }
      />
      <StatStrip
        items={[
          { label: intl.formatMessage({ id: "review.stat.proposed" }), value: proposedCount },
          { label: intl.formatMessage({ id: "review.stat.contested" }), value: contestedCount },
          { label: intl.formatMessage({ id: "review.stat.total" }), value: rows.length },
        ]}
      />
      <FilterChips<Filter>
        value={filter}
        onChange={setFilter}
        items={[
          { id: "all", label: intl.formatMessage({ id: "review.filter.all" }), count: rows.length },
          {
            id: "proposed",
            label: intl.formatMessage({ id: "review.filter.proposed" }),
            count: proposedCount,
          },
          {
            id: "contested",
            label: intl.formatMessage({ id: "review.filter.contested" }),
            count: contestedCount,
          },
        ]}
      />
      {err ? <p className="err">{err}</p> : <p className="meta">{status}</p>}
      <div>
        {[...groups.entries()].map(([key, items]) => {
          const label =
            key === "_proposed"
              ? intl.formatMessage({ id: "review.group.proposed" })
              : key === "_contested_ungrouped"
                ? intl.formatMessage({ id: "review.group.contestedUngrouped" })
                : intl.formatMessage({ id: "review.group.conflict" }, { group: key });
          return (
            <div key={key}>
              <div className="group-label">{label}</div>
              {items.map((r) => (
                <div key={r.uid} className={`row ${r.trust || ""}`}>
                  <div>
                    <h3>
                      <a href={`/entry?uid=${encodeURIComponent(r.uid)}`}>{r.title}</a>{" "}
                      <TrustBadge trust={r.trust} />
                    </h3>
                    <p>
                      {[r.uid, r.exp_type, r.confidence != null ? `conf ${r.confidence}` : "", r.symptom?.slice(0, 80)]
                        .filter(Boolean)
                        .join(" · ")}
                    </p>
                  </div>
                  <div className="row-actions">
                    {r.conflict_group ? (
                      <Button
                        size="sm"
                        className="ghost"
                        onClick={() => nav(`/entry?uid=${encodeURIComponent(r.uid)}`)}
                      >
                        <FormattedMessage id="review.action.compare" />
                      </Button>
                    ) : null}
                    <Button size="sm" variant="ok" onClick={() => void approve(r.uid)}>
                      <FormattedMessage id="review.action.approve" />
                    </Button>
                    <Button size="sm" variant="bad" onClick={() => void reject(r.uid)}>
                      <FormattedMessage id="review.action.reject" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          );
        })}
      </div>
      {toast ? <Toast message={toast} onDone={() => setToast("")} /> : null}
    </>
  );
}
