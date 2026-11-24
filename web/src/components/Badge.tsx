import { FormattedMessage } from "react-intl";

const known = new Set(["active", "proposed", "contested", "stale", "archived", "falsified"]);

export function TrustBadge({ trust }: { trust?: string | null }) {
  const t = (trust || "unknown").toLowerCase();
  const cls = known.has(t) ? t : "";
  if (known.has(t)) {
    return (
      <span className={`badge ${cls}`}>
        <FormattedMessage id={`entry.trust.${t}`} />
      </span>
    );
  }
  return <span className="badge">{trust || "—"}</span>;
}
