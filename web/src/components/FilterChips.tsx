export type ChipItem<T extends string> = {
  id: T;
  label: string;
  count?: number;
};

export function FilterChips<T extends string>({
  items,
  value,
  onChange,
}: {
  items: ChipItem<T>[];
  value: T;
  onChange: (id: T) => void;
}) {
  return (
    <div className="filter-chips" role="tablist">
      {items.map((it) => (
        <button
          key={it.id}
          type="button"
          role="tab"
          aria-selected={value === it.id}
          className={`chip ${value === it.id ? "on" : ""}`}
          onClick={() => onChange(it.id)}
        >
          {it.label}
          {it.count != null ? <span className="count">{it.count}</span> : null}
        </button>
      ))}
    </div>
  );
}
