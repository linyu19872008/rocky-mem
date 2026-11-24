import { FormattedMessage } from "react-intl";
import { PageHeader } from "../components/PageHeader";

export function IngestPage() {
  return (
    <>
      <PageHeader title={<FormattedMessage id="ingest.title" />} />
      <div className="drop-zone">
        <div style={{ fontWeight: 650, marginBottom: "0.5rem" }}>
          <FormattedMessage id="ingest.drop" />
        </div>
        <p className="meta" style={{ margin: 0 }}>
          <FormattedMessage id="ingest.hint" />
        </p>
      </div>
    </>
  );
}
