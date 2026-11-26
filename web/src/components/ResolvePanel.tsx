import { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { api, ApiRequestError } from "../lib/api";
import { Button } from "./Button";
import { DataCard } from "./DataCard";

type Related = {
  uid: string;
  title?: string;
  symptom?: string;
  fix_md?: string;
  applies_when?: string;
  invalid_when?: string;
};

type Actions = "adoptThis" | "adoptPeer" | "split" | "reject_both" | "defer";

const ORDER: Actions[] = ["adoptThis", "adoptPeer", "split", "reject_both", "defer"];

const LABEL: Record<Actions, string> = {
  adoptThis: "resolve.action.adoptThis",
  adoptPeer: "resolve.action.adoptPeer",
  split: "resolve.action.split",
  reject_both: "resolve.action.rejectBoth",
  defer: "resolve.action.defer",
};

export function ResolvePanel(props: {
  uid: string;
  entry: Record<string, unknown>;
  peer?: Related;
  onDone: () => void | Promise<void>;
}) {
  const intl = useIntl();
  const { uid, entry, peer, onDone } = props;
  const [action, setAction] = useState<Actions>("adoptThis");
  const [note, setNote] = useState("");
  const [days, setDays] = useState(30);
  const [selfApplies, setSelfApplies] = useState(String(entry.applies_when || ""));
  const [selfInvalid, setSelfInvalid] = useState(String(entry.invalid_when || ""));
  const [peerApplies, setPeerApplies] = useState(String(peer?.applies_when || ""));
  const [peerInvalid, setPeerInvalid] = useState(String(peer?.invalid_when || ""));
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    setSelfApplies(String(entry.applies_when || ""));
    setSelfInvalid(String(entry.invalid_when || ""));
    setPeerApplies(String(peer?.applies_when || ""));
    setPeerInvalid(String(peer?.invalid_when || ""));
  }, [uid, entry.applies_when, entry.invalid_when, peer?.applies_when, peer?.invalid_when]);

  async function submit() {
    if (busy) return;
    setBusy(true);
    setErr("");
    try {
      const body: Record<string, unknown> = { uid, note };
      if (action === "adoptThis") {
        body.action = "adopt";
        body.keep = uid;
      } else if (action === "adoptPeer") {
        body.action = "adopt";
        body.keep = peer?.uid;
      } else if (action === "split") {
        body.action = "split";
        body.keep_applies_when = selfApplies;
        body.keep_invalid_when = selfInvalid;
        body.peer_applies_when = peerApplies;
        body.peer_invalid_when = peerInvalid;
      } else if (action === "reject_both") {
        body.action = "reject_both";
      } else {
        body.action = "defer";
        body.review_after_days = days;
      }
      await api("/api/v1/review/resolve", body);
      setNote("");
      await onDone();
    } catch (ex) {
      const code = ex instanceof ApiRequestError ? ex.code : "ERR_SYS_INTERNAL";
      setErr(
        ex instanceof ApiRequestError && ex.message
          ? ex.message
          : intl.formatMessage({ id: `error.${code}`, defaultMessage: intl.formatMessage({ id: "error.generic" }) }),
      );
    } finally {
      setBusy(false);
    }
  }

  // 1-5 pick an action; Ctrl/Cmd+Enter submits. The adjudication panel is the
  // one screen where a mouse round-trip really costs.
  useEffect(() => {
    function onKey(ev: KeyboardEvent) {
      const tag = (ev.target as HTMLElement)?.tagName;
      const typing = tag === "INPUT" || tag === "TEXTAREA";
      if ((ev.metaKey || ev.ctrlKey) && ev.key === "Enter") {
        ev.preventDefault();
        void submit();
        return;
      }
      if (typing) return;
      const n = Number(ev.key);
      if (n >= 1 && n <= ORDER.length) {
        ev.preventDefault();
        setAction(ORDER[n - 1]);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const showSplit = action === "split";
  const showDays = action === "defer";

  return (
    <DataCard title={<FormattedMessage id="resolve.title" />}>
      {err ? <p className="err">{err}</p> : null}
      {peer ? (
        <p className="meta">
          <FormattedMessage id="resolve.peer" />: <strong>{peer.title || peer.uid}</strong>
          {peer.symptom ? ` — ${peer.symptom}` : null}
        </p>
      ) : null}

      <div className="resolve-actions">
        {ORDER.map((a, i) => (
          <button
            key={a}
            type="button"
            className={`resolve-action${action === a ? " is-on" : ""}`}
            onClick={() => setAction(a)}
            disabled={a === "adoptPeer" && !peer}
          >
            <span className="meta">{i + 1}</span> <FormattedMessage id={LABEL[a]} />
          </button>
        ))}
      </div>

      {showSplit ? (
        <div className="resolve-conditions">
          <label>
            <FormattedMessage id="resolve.cond.selfApplies" />
            <input value={selfApplies} onChange={(e) => setSelfApplies(e.target.value)} />
          </label>
          <label>
            <FormattedMessage id="resolve.cond.selfInvalid" />
            <input value={selfInvalid} onChange={(e) => setSelfInvalid(e.target.value)} />
          </label>
          <label>
            <FormattedMessage id="resolve.cond.peerApplies" />
            <input value={peerApplies} onChange={(e) => setPeerApplies(e.target.value)} />
          </label>
          <label>
            <FormattedMessage id="resolve.cond.peerInvalid" />
            <input value={peerInvalid} onChange={(e) => setPeerInvalid(e.target.value)} />
          </label>
          <p className="meta">
            <FormattedMessage id="resolve.cond.hint" />
          </p>
        </div>
      ) : null}

      {showDays ? (
        <label className="resolve-days">
          <FormattedMessage id="resolve.days" />
          <input
            type="number"
            min={1}
            value={days}
            onChange={(e) => setDays(Number(e.target.value) || 30)}
          />
        </label>
      ) : null}

      <label className="resolve-note">
        <FormattedMessage id="resolve.note" />
        <textarea
          rows={3}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={intl.formatMessage({ id: "resolve.notePlaceholder" })}
        />
      </label>

      <div className="ingest-actions" style={{ border: 0, paddingTop: 0, justifyContent: "flex-start" }}>
        <Button variant="primary" disabled={busy || note.trim() === ""} onClick={() => void submit()}>
          <FormattedMessage id="resolve.submit" />
        </Button>
        <span className="meta">
          <FormattedMessage id="resolve.shortcut" />
        </span>
      </div>
    </DataCard>
  );
}
