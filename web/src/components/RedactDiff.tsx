import type { ReactNode } from "react";

export type DiffSegment = {
  kind: "eq" | "del" | "ins" | string;
  text: string;
  rule?: string;
};

export function RedactDiff({ segments }: { segments: DiffSegment[] }) {
  const nodes: ReactNode[] = [];
  for (let i = 0; i < segments.length; i++) {
    const s = segments[i];
    if (s.kind === "eq") {
      nodes.push(
        <span key={i} className="diff-eq">
          {s.text}
        </span>,
      );
      continue;
    }
    if (s.kind === "del") {
      nodes.push(
        <del key={i} className="diff-del" title={s.rule || "redacted"}>
          {s.text}
        </del>,
      );
      continue;
    }
    if (s.kind === "ins") {
      nodes.push(
        <ins key={i} className="diff-ins" title={s.rule || "replacement"}>
          {s.text}
        </ins>,
      );
    }
  }
  return (
    <pre className="redact-diff mono" aria-label="redaction diff">
      {nodes}
    </pre>
  );
}
