package api

import (
	"encoding/json"

	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/version"
)

// doctorPayload decorates the store report with what a client needs to decide
// whether it can use this server: the contract generation, the release string,
// and the features actually available. Feature names are the same strings a
// client probes (for example "context" or "mcp.streamable-http").
func doctorPayload(rep *store.DoctorReport) map[string]any {
	payload := map[string]any{}
	if raw, err := json.Marshal(rep); err == nil {
		_ = json.Unmarshal(raw, &payload)
	}

	features := append([]string{}, Features...)
	if rep.FTSAvailable {
		features = append(features, "retrieval.fts")
	}

	payload["contract"] = ContractVersion
	payload["version"] = version.Version
	payload["features"] = features
	payload["mcp"] = map[string]any{
		"streamable_http": true,
		"stateless":       true,
	}
	return payload
}
