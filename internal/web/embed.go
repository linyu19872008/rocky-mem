// Package web serves the minimal search/review UI embedded in rockymem serve.
package web

import (
	"embed"
	"html/template"
	"net/http"
)

// SessionCookie is the HttpOnly cookie used by the loopback web UI.
const SessionCookie = "rocky_mem_session"

//go:embed templates/*.html
var templatesFS embed.FS

type pageData struct {
	Title string
}

// SetSessionCookie writes the loopback web session cookie.
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// Mount registers unauthenticated HTML routes and sets the session cookie so
// same-origin fetch to /api/v1 works without a manual Bearer token.
func Mount(mux *http.ServeMux, token string) {
	indexT := template.Must(template.ParseFS(templatesFS, "templates/index.html"))
	reviewT := template.Must(template.ParseFS(templatesFS, "templates/review.html"))
	entryT := template.Must(template.ParseFS(templatesFS, "templates/entry.html"))
	settingsT := template.Must(template.ParseFS(templatesFS, "templates/settings.html"))

	serve := func(t *template.Template, title string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			SetSessionCookie(w, token)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = t.Execute(w, pageData{Title: title})
		}
	}

	mux.HandleFunc("GET /{$}", serve(indexT, "rocky-mem search"))
	mux.HandleFunc("GET /review", serve(reviewT, "rocky-mem review"))
	mux.HandleFunc("GET /entry", serve(entryT, "rocky-mem entry"))
	mux.HandleFunc("GET /settings", serve(settingsT, "rocky-mem settings"))
}
