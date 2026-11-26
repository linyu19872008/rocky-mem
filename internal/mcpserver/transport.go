package mcpserver

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StreamableHandler serves the MCP Streamable HTTP transport.
//
// It runs stateless: every request carries its own protocol revision and no
// session is created, so there is no server-side session table, no sticky
// routing, and a restart loses nothing. GET and DELETE answer 405 by design —
// rocky-mem exposes request/response tools only, with no server push.
func StreamableHandler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)
}
