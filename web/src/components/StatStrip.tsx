export type StatItem = { label: string; value: string | number };

export function StatStrip({ items }: { items: StatItem[] }) {
  return (
    <div className="stat-strip">
      {items.map((it) => (
        <div className="stat-card" key={it.label}>
          <div className="label">{it.label}</div>
          <div className="value">{it.value}</div>
        </div>
      ))}
    </div>
  );
}
