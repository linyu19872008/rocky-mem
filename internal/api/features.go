package api

// ContractVersion is the client-facing contract generation. Additive changes
// keep it; removing or reshaping a tool, endpoint, field, or error code bumps it.
const ContractVersion = 1

// Features is the single list every client-facing capability is drawn from.
// `GET /api/v1/doctor` publishes it verbatim, and docs/contract.md must mention
// each entry — a CI test asserts that, so a capability cannot be announced in
// one place and forgotten in another.
//
// Adding a feature is additive: append it, implement it, document it.
var Features = []string{
	// REST surface, one entry per user-visible operation group.
	"context",          // POST /api/v1/context
	"search",           // POST /api/v1/search
	"get",              // POST /api/v1/get
	"remember",         // POST /api/v1/remember
	"record",           // POST /api/v1/record
	"feedback",         // POST /api/v1/feedback
	"browse",           // POST /api/v1/browse
	"review",           // POST /api/v1/review/*
	"corpus",           // source/chunk storage and the Background section
	"conflict.resolve", // PUT /api/v1/review/resolve

	// Transports.
	"mcp.stdio",           // rockymem mcp
	"mcp.streamable-http", // POST /mcp
}
