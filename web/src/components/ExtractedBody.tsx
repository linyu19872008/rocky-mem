import { useMemo, useState } from "react";
import { FormattedMessage } from "react-intl";

type Block =
  | { kind: "text"; text: string }
  | { kind: "table"; rows: string[][] };

function splitBlocks(text: string): Block[] {
  const lines = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  const blocks: Block[] = [];
  let textBuf: string[] = [];
  let tableBuf: string[][] = [];

  const flushText = () => {
    if (!textBuf.length) return;
    const t = textBuf.join("\n").trimEnd();
    if (t.trim()) blocks.push({ kind: "text", text: t });
    textBuf = [];
  };
  const flushTable = () => {
    if (!tableBuf.length) return;
    blocks.push({ kind: "table", rows: tableBuf });
    tableBuf = [];
  };

  for (const line of lines) {
    if (line.includes("\t")) {
      flushText();
      tableBuf.push(line.split("\t"));
      continue;
    }
    flushTable();
    textBuf.push(line);
  }
  flushTable();
  flushText();
  return blocks;
}

export function ExtractedBody({ text, className }: { text: string; className?: string }) {
  const [expanded, setExpanded] = useState(false);
  const blocks = useMemo(() => splitBlocks(text), [text]);
  const hasTable = blocks.some((b) => b.kind === "table");
  const long = text.length > 1800 || text.split("\n").length > 28;

  return (
    <div className={`extracted-body ${expanded ? "is-expanded" : ""} ${className || ""}`}>
      <div className="extracted-scroll">
        {hasTable ? (
          blocks.map((b, i) =>
            b.kind === "text" ? (
              <pre key={i} className="extracted-pre mono extracted-pre-inline">
                {b.text}
              </pre>
            ) : (
              <table key={i} className="extracted-table">
                <tbody>
                  {b.rows.map((row, ri) => (
                    <tr key={ri} className={ri === 0 ? "is-head" : undefined}>
                      {row.map((cell, ci) => (
                        <td key={ci}>{cell}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            ),
          )
        ) : (
          <pre className="extracted-pre mono">{text}</pre>
        )}
      </div>
      {long ? (
        <button type="button" className="extracted-expand" onClick={() => setExpanded((v) => !v)}>
          <FormattedMessage id={expanded ? "ingest.collapse" : "ingest.expand"} />
        </button>
      ) : null}
    </div>
  );
}
