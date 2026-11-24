import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { Button } from "../components/Button";
import { TrustBadge } from "../components/Badge";
import { Banner } from "../components/Banner";
import { DataCard } from "../components/DataCard";
import { PageHeader } from "../components/PageHeader";

type Related = {
  uid: string;
  relation?: string;
  title?: string;
  trust?: string;
  symptom?: string;
  fix_md?: string;
  applies_when?: string;
  invalid_when?: string;
  outcome?: string;
  confidence?: number;
};

type GetResult = {
  type: string;
  uid: string;
  entry: Record<string, unknown>;
  evidence?: Record<string, unknown>[];
  related?: Related[];
};

function field(label: string, val: unknown) {
  if (val === undefined || val === null || val === "") return null;
  return (
    <div className="field">
      <b>{label}:</b> {String(val)}
    </div>
  );
}

function Col({
  title,
  href,
  trust,
  fields,
  cls,
}: {
  title: string;
  href?: string;
  trust?: string;
  fields: [string, unknown][];
  cls: string;
}) {
  return (
    <div className={`col ${cls}`}>
      <h3>
        {href ? <Link to={href}>{title}</Link> : title} <TrustBadge trust={trust} />
      </h3>
      {fields.map(([label, val]) =>
        val === undefined || val === null || val === "" ? null : (
          <div key={label}>
            <div className="label">{label}</div>
            <pre>{String(val)}</pre>
          </div>
        ),
      )}
    </div>
  );
}

export function EntryPage() {
  const intl = useIntl();
  const [params, setParams] = useSearchParams();
  const uid = (params.get("uid") || "").trim();
  const [status, setStatus] = useState("");
  const [err, setErr] = useState("");
  const [data, setData] = useState<GetResult | null>(null);
  const [note, setNote] = useState("");

  const load = useCallback(async () => {
    if (!uid) {
      setData(null);
      setStatus(intl.formatMessage({ id: "entry.openPrompt" }));
      return;
    }
    setErr("");
    setStatus(intl.formatMessage({ id: "entry.loading" }, { uid }));
    try {
      const got = await api<GetResult>("/api/v1/get", { uid, include_raw: true });
      setData(got);
      setStatus(`${got.type} · ${got.uid}`);
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setErr(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
      setData(null);
    }
  }, [uid, intl]);

  useEffect(() => {
    void load();
  }, [load]);

  function onOpen(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const v = new FormData(e.currentTarget).get("uid");
    if (typeof v === "string" && v.trim()) setParams({ uid: v.trim() });
  }

  const e = data?.entry || {};
  const trust = String(e.trust || e.status || "");
  const title = String(e.title || data?.uid || intl.formatMessage({ id: "entry.title" }));
  const lang = typeof e.lang === "string" ? e.lang : undefined;
  const canReview = data?.type === "experience" && (trust === "proposed" || trust === "contested");

  const headerActions = canReview ? (
    <>
      <Button
        variant="ok"
        onClick={async () => {
          await api("/api/v1/review/approve", { uid: data!.uid });
          await load();
        }}
      >
        <FormattedMessage id="review.action.approve" />
      </Button>
      <Button
        variant="bad"
        onClick={async () => {
          await api("/api/v1/review/reject", { uid: data!.uid, reason: "web-reject" });
          await load();
        }}
      >
        <FormattedMessage id="review.action.reject" />
      </Button>
    </>
  ) : undefined;

  if (!uid) {
    return (
      <>
        <PageHeader title={<FormattedMessage id="entry.title" />} />
        <p className="meta">{status}</p>
        <form className="search-field" onSubmit={onOpen}>
          <input name="uid" placeholder={intl.formatMessage({ id: "entry.openPlaceholder" })} />
          <Button type="submit" variant="primary">
            <FormattedMessage id="entry.openSubmit" />
          </Button>
        </form>
      </>
    );
  }

  return (
    <>
      <PageHeader
        title={
          <span style={{ display: "inline-flex", gap: "0.5rem", alignItems: "center", flexWrap: "wrap" }}>
            <span lang={lang}>{title}</span>
            {data ? <TrustBadge trust={trust} /> : null}
          </span>
        }
        actions={headerActions}
      />
      {err ? <p className="err">{err}</p> : <p className="meta">{status}</p>}
      {data ? (
        <>
          {trust === "contested" ? (
            <Banner kind="contested">
              <FormattedMessage id="entry.banner.contested" />
            </Banner>
          ) : null}
          {trust === "proposed" ? (
            <Banner kind="proposed">
              <FormattedMessage id="entry.banner.proposed" />
            </Banner>
          ) : null}

          <DataCard>
            {field(intl.formatMessage({ id: "entry.field.type" }), e.exp_type || e.mem_type)}
            {field(
              intl.formatMessage({ id: "entry.field.scope" }),
              `${e.scope || ""}${e.scope_ref ? "/" + e.scope_ref : ""}`,
            )}
            {field(intl.formatMessage({ id: "entry.field.confidence" }), e.confidence)}
            {field(intl.formatMessage({ id: "entry.field.outcome" }), e.outcome)}
            {field(intl.formatMessage({ id: "entry.field.appliesWhen" }), e.applies_when)}
            {field(intl.formatMessage({ id: "entry.field.invalidWhen" }), e.invalid_when)}
            {field(intl.formatMessage({ id: "entry.field.symptom" }), e.symptom)}
            {field(intl.formatMessage({ id: "entry.field.rootCause" }), e.root_cause)}
            {field(intl.formatMessage({ id: "entry.field.conflictGroup" }), e.conflict_group)}
          </DataCard>

          {e.fix_md ? (
            <DataCard title={<FormattedMessage id="entry.fix" />}>
              <pre lang={lang}>{String(e.fix_md)}</pre>
            </DataCard>
          ) : null}
          {e.body_md ? (
            <DataCard title={<FormattedMessage id="entry.body" />}>
              <pre lang={lang}>{String(e.body_md)}</pre>
            </DataCard>
          ) : null}

          {data.related && data.related.length > 0 ? (
            <DataCard title={<FormattedMessage id="entry.compare" />}>
              <div className="compare">
                <Col
                  title={title}
                  trust={trust}
                  cls="current"
                  fields={[
                    ["symptom", e.symptom],
                    ["fix", e.fix_md],
                    ["applies_when", e.applies_when],
                    ["invalid_when", e.invalid_when],
                    ["outcome", e.outcome],
                    ["confidence", e.confidence],
                  ]}
                />
                {data.related.map((r) => (
                  <Col
                    key={r.uid}
                    title={r.title || r.uid}
                    href={`/entry?uid=${encodeURIComponent(r.uid)}`}
                    trust={r.trust}
                    cls="peer"
                    fields={[
                      ["symptom", r.symptom],
                      ["fix", r.fix_md],
                      ["applies_when", r.applies_when],
                      ["invalid_when", r.invalid_when],
                      ["outcome", r.outcome],
                      ["confidence", r.confidence],
                      ["relation", r.relation],
                    ]}
                  />
                ))}
              </div>
            </DataCard>
          ) : null}

          {data.evidence && data.evidence.length > 0 ? (
            <DataCard title={<FormattedMessage id="entry.evidence" />}>
              {data.evidence.map((ev, i) => (
                <div className="field" key={i}>
                  <b>{String(ev.role || "evidence")}</b>
                  {ev.channel ? ` · ${String(ev.channel)}` : ""}
                  {ev.quote || ev.excerpt ? <pre>{String(ev.quote || ev.excerpt)}</pre> : null}
                </div>
              ))}
            </DataCard>
          ) : null}

          <DataCard title={<FormattedMessage id="entry.rawJson" />}>
            <pre style={{ maxHeight: "16rem" }}>{JSON.stringify(data, null, 2)}</pre>
          </DataCard>

          <DataCard title={<FormattedMessage id="entry.feedback" />}>
            <textarea
              value={note}
              onChange={(ev) => setNote(ev.target.value)}
              placeholder={intl.formatMessage({ id: "entry.feedbackNote" })}
              style={{ width: "100%", minHeight: "4rem" }}
            />
            <div className="row-actions" style={{ marginBlockStart: "0.5rem" }}>
              {(["helpful", "wrong", "stale"] as const).map((outcome) => (
                <Button
                  key={outcome}
                  variant={outcome === "wrong" ? "warn" : "default"}
                  onClick={async () => {
                    await api("/api/v1/feedback", { uid: data.uid, outcome, note });
                    await load();
                  }}
                >
                  <FormattedMessage id={`entry.feedback.${outcome}`} />
                </Button>
              ))}
            </div>
          </DataCard>
        </>
      ) : null}
    </>
  );
}
