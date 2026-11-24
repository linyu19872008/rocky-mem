package api

import (
	"net/http"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/web"
)

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Token != "" && !s.authorized(r) {
			writeErr(w, http.StatusUnauthorized, errs.New(errs.Unauthorized, "missing or invalid bearer token", nil))
			return
		}
		next(w, r)
	}
}

func (s *Server) authorized(r *http.Request) bool {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		if strings.TrimPrefix(auth, "Bearer ") == s.Token {
			return true
		}
	}
	if c, err := r.Cookie(web.SessionCookie); err == nil && c.Value == s.Token {
		return true
	}
	return false
}
