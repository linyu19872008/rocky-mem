import { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { apiGet, ApiRequestError } from "../lib/api";
import { DataCard } from "../components/DataCard";
import { LocaleSelect } from "../components/LocaleSelect";
import { ThemeToggle } from "../components/ThemeToggle";
import { PageHeader } from "../components/PageHeader";

export function SettingsPage() {
  const intl = useIntl();
  const [status, setStatus] = useState(intl.formatMessage({ id: "settings.doctorLoading" }));
  const [ok, setOk] = useState<boolean | null>(null);
  const [doctor, setDoctor] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const data = await apiGet<Record<string, unknown>>("/api/v1/doctor");
        const good = data.search_doc_ok !== false;
        setOk(good);
        setStatus(
          good
            ? intl.formatMessage({ id: "settings.doctorOk" })
            : intl.formatMessage({ id: "settings.doctorBad" }),
        );
        setDoctor(JSON.stringify(data, null, 2));
      } catch (ex) {
        const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
        setOk(false);
        setStatus(intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }));
        setDoctor("");
      }
    })();
  }, [intl]);

  return (
    <>
      <PageHeader title={<FormattedMessage id="settings.title" />} />
      <p className={ok === false ? "err" : "meta"} style={ok ? { color: "var(--ok)" } : undefined}>
        {status}
      </p>
      <DataCard title={<FormattedMessage id="settings.prefs" />}>
        <p className="meta" style={{ marginTop: 0 }}>
          <FormattedMessage id="settings.prefsHint" />
        </p>
        <div style={{ display: "flex", gap: "1rem", alignItems: "center", flexWrap: "wrap" }}>
          <span className="meta">
            <FormattedMessage id="locale.label" />
          </span>
          <LocaleSelect />
          <span className="meta">
            <FormattedMessage id="theme.label" />
          </span>
          <ThemeToggle />
        </div>
      </DataCard>
      <DataCard title={<FormattedMessage id="settings.health" />}>
        <pre className="mono">{doctor || "…"}</pre>
      </DataCard>
      <DataCard title={<FormattedMessage id="settings.ops" />}>
        <ul className="meta">
          <li>
            <FormattedMessage id="settings.ops.backup" />
          </li>
          <li>
            <FormattedMessage id="settings.ops.cron" />
          </li>
          <li>
            <FormattedMessage id="settings.ops.auth" />
          </li>
        </ul>
      </DataCard>
    </>
  );
}
