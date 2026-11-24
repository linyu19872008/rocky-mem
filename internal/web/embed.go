// Package web serves the minimal search/review UI embedded in rockymem serve.
package web

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templatesFS embed.FS

type pageData struct {
	Token string
	Title string
}

// Mount registers unauthenticated HTML routes. API calls from the page use the injected token.
func Mount(mux *http.ServeMux, token string) {
	indexT := template.Must(template.ParseFS(templatesFS, "templates/index.html"))
	reviewT := template.Must(template.ParseFS(templatesFS, "templates/review.html"))

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = indexT.Execute(w, pageData{Token: token, Title: "rocky-mem search"})
	})
	mux.HandleFunc("GET /review", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = reviewT.Execute(w, pageData{Token: token, Title: "rocky-mem review"})
	})
}
