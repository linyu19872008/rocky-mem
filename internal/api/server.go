// Package api exposes the loopback HTTP /api/v1 surface used by serve and clients.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/web"
)

// Server wraps core.Service with bearer-token auth.
type Server struct {
	Svc       *core.Service
	Token     string
	EnableWeb bool
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.auth(s.handleHealth))
	mux.HandleFunc("GET /api/v1/doctor", s.auth(s.handleDoctor))
	mux.HandleFunc("GET /api/v1/stats", s.auth(s.handleStats))
	mux.HandleFunc("POST /api/v1/context", s.auth(s.handleContext))
	mux.HandleFunc("POST /api/v1/search", s.auth(s.handleSearch))
	mux.HandleFunc("POST /api/v1/get", s.auth(s.handleGet))
	mux.HandleFunc("POST /api/v1/remember", s.auth(s.handleRemember))
	mux.HandleFunc("POST /api/v1/record", s.auth(s.handleRecord))
	mux.HandleFunc("POST /api/v1/feedback", s.auth(s.handleFeedback))
	mux.HandleFunc("POST /api/v1/review/list", s.auth(s.handleReviewList))
	mux.HandleFunc("POST /api/v1/review/approve", s.auth(s.handleApprove))
	mux.HandleFunc("POST /api/v1/review/reject", s.auth(s.handleReject))
	mux.HandleFunc("POST /api/v1/reindex", s.auth(s.handleReindex))
	mux.HandleFunc("POST /api/v1/purge", s.auth(s.handlePurge))
	mux.HandleFunc("POST /api/v1/export", s.auth(s.handleExport))
	mux.HandleFunc("POST /api/v1/backup", s.auth(s.handleBackup))
	if s.EnableWeb {
		web.Mount(mux, s.Token)
	}
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	rep, err := s.Svc.Store.Doctor(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	out, err := s.Svc.Stats(r.Context())
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	var in core.ContextInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Context(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var in core.SearchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Search(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	var in core.GetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Get(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRemember(w http.ResponseWriter, r *http.Request) {
	var in core.RememberInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Remember(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request) {
	var in core.RecordInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Record(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	var in core.FeedbackInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Feedback(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleReviewList(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Trusts []string `json:"trusts"`
		Limit  int      `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		// Empty body is fine — defaults apply.
		if err.Error() != "EOF" {
			writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
			return
		}
	}
	out, err := s.Svc.ReviewList(r.Context(), in.Trusts, in.Limit)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UID string `json:"uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.ReviewApprove(r.Context(), in.UID)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UID    string `json:"uid"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.ReviewReject(r.Context(), in.UID, in.Reason)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	out, err := s.Svc.Reindex(r.Context())
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePurge(w http.ResponseWriter, r *http.Request) {
	var in core.PurgeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Purge(r.Context(), in)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Format string `json:"format"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	out, err := s.Svc.ExportL1(r.Context(), in.Format)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, errs.New(errs.InvalidRequest, err.Error(), nil))
		return
	}
	out, err := s.Svc.Backup(r.Context(), in.Path)
	if err != nil {
		writeErr(w, statusOf(err), errs.As(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, e *errs.E) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": e})
}

func statusOf(err error) int {
	e := errs.As(err)
	switch e.Code {
	case errs.Unauthorized:
		return http.StatusUnauthorized
	case errs.Forbidden, errs.ScopeForbidden:
		return http.StatusForbidden
	case errs.NotFound:
		return http.StatusNotFound
	case errs.WriteConflict:
		return http.StatusConflict
	case errs.InvalidRequest, errs.FieldRequired, errs.FieldInvalid, errs.EvidenceRequired, errs.PolicyRejected:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
