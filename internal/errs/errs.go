// Package errs defines stable public error codes (docs/error-codes.md).
package errs

import (
	"encoding/json"
	"fmt"
)

// Code is a stable machine-readable error identifier.
type Code string

const (
	Unauthorized      Code = "ERR_UNAUTHORIZED"
	Forbidden         Code = "ERR_FORBIDDEN"
	ScopeForbidden    Code = "ERR_SCOPE_FORBIDDEN"
	InvalidRequest    Code = "ERR_INVALID_REQUEST"
	FieldRequired     Code = "ERR_FIELD_REQUIRED"
	FieldInvalid      Code = "ERR_FIELD_INVALID"
	UnsupportedLang   Code = "ERR_UNSUPPORTED_LANG"
	PayloadTooLarge   Code = "ERR_PAYLOAD_TOO_LARGE"
	EvidenceRequired  Code = "ERR_EVIDENCE_REQUIRED"
	WriteConflict     Code = "ERR_WRITE_CONFLICT"
	NotFound          Code = "ERR_NOT_FOUND"
	ProjectUnresolved Code = "ERR_PROJECT_UNRESOLVED"
	PolicyRejected    Code = "ERR_POLICY_REJECTED"
	FTSUnavailable    Code = "ERR_FTS_UNAVAILABLE"
	DBError           Code = "ERR_DB"
	Internal          Code = "ERR_SYS_INTERNAL"
)

// E is a structured application error.
type E struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params,omitempty"`
}

func (e *E) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New constructs an error with optional params.
func New(code Code, message string, params map[string]any) *E {
	return &E{Code: code, Message: message, Params: params}
}

// JSON serialises the error envelope for MCP/HTTP clients.
func (e *E) JSON() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// As returns e if err is *E, otherwise wraps as Internal.
func As(err error) *E {
	if err == nil {
		return nil
	}
	if e, ok := err.(*E); ok {
		return e
	}
	return New(Internal, err.Error(), nil)
}
