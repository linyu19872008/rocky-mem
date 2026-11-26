# Client contract

Two clients ship today — the DSH bundle and Cursor — and both reach the server
over the same two surfaces: the REST API and MCP. This page is the compatibility
contract they may rely on. Anything not listed here is an implementation detail.

## Versioning

`GET /api/v1/doctor` returns `contract`. The rules:

| Change | Contract |
|---|---|
| New endpoint, new tool, new optional field, new feature entry | unchanged (additive) |
| New error code | unchanged |
| Removing or renaming an endpoint, tool, or field; changing a field's type or meaning; changing a default that alters behaviour | **+1** |

A client must read `features` rather than infer capability from the release
string. A client that sees a `contract` above what it supports should degrade to
read-only rather than guess.

## Capability list

Every entry below is published in `doctor.features` and is asserted against this
page by a test, so the list cannot drift from the documentation.

| Feature | Surface | Meaning |
|---|---|---|
| `context` | `POST /api/v1/context` | standing memory pack for a working directory |
| `search` | `POST /api/v1/search` | budgeted, provenance-backed context pack |
| `get` | `POST /api/v1/get` | one entry with its evidence chain and neighbours |
| `remember` | `POST /api/v1/remember` | upsert a memory (current value per key) |
| `record` | `POST /api/v1/record` | append an experience claim (evidence required) |
| `feedback` | `POST /api/v1/feedback` | helpful / wrong / stale |
| `browse` | `POST /api/v1/browse` | paginated listing for UIs |
| `review` | `POST /api/v1/review/*` | the pending-review queue |
| `corpus` | `source` / `chunk` storage | long material returned in `## Background` |
| `conflict.resolve` | `POST /api/v1/review/resolve` | adjudicate a contested pair |
| `mcp.stdio` | `rockymem mcp` | MCP over stdio |
| `mcp.streamable-http` | `POST /mcp` | MCP over Streamable HTTP, stateless |

## Tools

Six MCP tools, identical on both transports:
`mem_context`, `mem_search`, `mem_get`, `mem_record`, `mem_remember`,
`mem_feedback`. Their names and required parameters are mirrored in
`skills/rocky-mem/tools.json` and checked against `internal/mcpserver` by
`scripts/check-skill-versions.mjs`.

The context pack keeps four fixed English headings — `## Preferences`,
`## Applies`, `## Do not use`, `## Background` — so clients and prompts can rely
on structure. The body stays in the language the entry was written in.

## Error codes

`{code, message, params}` with stable codes from `internal/errs/errs.go`;
`docs/error-codes.md` is the registry. Codes a client may branch on are the ones
listed there, never the message text.

## Auth

`Authorization: Bearer <token>` on every REST route and on `/mcp`. The token is
written next to the database (`<db-stem>.token`, mode 600) or under
`$ROCKYMEM_HOME/pg-<hash>/`. `ROCKYMEM_URL` and `ROCKYMEM_TOKEN` override
discovery for clients that cannot read those files.

## Compatibility notes

- `GET /mcp` and `DELETE /mcp` answer `405`: the HTTP transport is stateless and
  has no server-initiated notifications. Clients that probe with `GET` must fall
  back to `POST`.
- `kinds: ["chunk"]` is supported; asking for an unknown kind returns
  `ERR_FIELD_INVALID`.
- `mem_search` includes `proposed` entries by default (`include_proposed: true`)
  at reduced rank. A client that needs authoritative only must pass `false`.
