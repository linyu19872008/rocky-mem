import type { ReactNode } from "react";

export function Panel({ title, children }: { title?: ReactNode; children: ReactNode }) {
  return (
    <div className="panel">
      {title ? <h2>{title}</h2> : null}
      {children}
    </div>
  );
}
