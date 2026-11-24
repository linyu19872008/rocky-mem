import type { ReactNode } from "react";

export function DataCard({ title, children, className = "" }: { title?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <div className={`data-card ${className}`.trim()}>
      {title ? <h2>{title}</h2> : null}
      {children}
    </div>
  );
}
