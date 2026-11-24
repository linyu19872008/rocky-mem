import { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { api, apiGet, ApiRequestError } from "../lib/api";
import { DataCard } from "../components/DataCard";
import { LocaleSelect } from "../components/LocaleSelect";
import { ThemeToggle } from "../components/ThemeToggle";
import { PageHeader } from "../components/PageHeader";
import { Button } from "../components/Button";

type BackupResult = {
  path: string;
  bytes: number;
  created_at: string;
  pruned?: string[];
  retain_keep?: number;
};

export function SettingsPage() {
  const intl = useIntl();
  const [status, setStatus] = useState(intl.formatMessage({ id: "settings.doctorLoading" }));
  const [ok, setOk] = useState<boolean | null>(null);
  const [doctor, setDoctor] = useState("");
  const [backupBusy, setBackupBusy] = useState(false);
  const [backupMsg, setBackupMsg] = useState("");
  const [backupErr, setBackupErr] = useState("");

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

  async function runBackup() {
    setBackupBusy(true);
    setBackupErr("");
    setBackupMsg("");
    try {
      const out = await api<BackupResult>("/api/v1/backup", { retain: 14 });
      const mb = (out.bytes / (1024 * 1024)).toFixed(2);
      setBackupMsg(
        intl.formatMessage(
          { id: "settings.backup.done" },
          { path: out.path, mb, at: out.created_at, pruned: out.pruned?.length ?? 0 },
        ),
      );
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_INTERNAL";
      setBackupErr(
        ex instanceof ApiRequestError && ex.message
          ? ex.message
          : intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }),
      );
    } finally {
      setBackupBusy(false);
    }
  }

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
        <p className="meta" style={{ marginTop: 0 }}>
          <FormattedMessage id="settings.backup.hint" />
        </p>
        <Button variant="primary" disabled={backupBusy} onClick={() => void runBackup()}>
          <FormattedMessage id={backupBusy ? "settings.backup.running" : "settings.backup.run"} />
        </Button>
        {backupErr ? <p className="err">{backupErr}</p> : null}
        {backupMsg ? <p className="meta" style={{ color: "var(--ok)" }}>{backupMsg}</p> : null}
        <ul className="meta" style={{ marginTop: "1rem" }}>
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
