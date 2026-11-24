import type { ReactNode } from "react";

export function Banner({ kind, children }: { kind: "proposed" | "contested"; children: ReactNode }) {
  return <div className={`banner ${kind}`}>{children}</div>;
}
