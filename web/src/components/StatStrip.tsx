export type StatItem = {
  label: string;
  value: string | number;
  href?: string;
  onClick?: () => void;
};

export function StatStrip({ items }: { items: StatItem[] }) {
  return (
    <div className="stat-strip">
      {items.map((it) => {
        const inner = (
          <>
            <div className="label">{it.label}</div>
            <div className="value">{it.value}</div>
          </>
        );
        if (it.href) {
          return (
            <a className="stat-card is-link" key={it.label} href={it.href} onClick={it.onClick}>
              {inner}
            </a>
          );
        }
        if (it.onClick) {
          return (
            <button type="button" className="stat-card is-link" key={it.label} onClick={it.onClick}>
              {inner}
            </button>
          );
        }
        return (
          <div className="stat-card" key={it.label}>
            {inner}
          </div>
        );
      })}
    </div>
  );
}
