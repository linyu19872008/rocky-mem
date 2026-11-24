// Package web serves the React SPA embedded in rockymem serve.
package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// SessionCookie is the HttpOnly cookie used by the loopback web UI.
const SessionCookie = "rocky_mem_session"

// LocaleCookie stores the user's UI locale preference.
const LocaleCookie = "rocky_mem_locale"

//go:embed all:dist
var distRoot embed.FS

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

type bootPayload struct {
	Locale    string `json:"locale"`
	ThemeHint string `json:"themeHint,omitempty"`
}

// Mount registers the SPA and sets the session cookie so same-origin fetch to
// /api/v1 works without a manual Bearer token.
func Mount(mux *http.ServeMux, token string) {
	root, err := fs.Sub(distRoot, "dist")
	if err != nil {
		panic("web: embed dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(root))

	mux.Handle("GET /assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetSessionCookie(w, token)
		fileServer.ServeHTTP(w, r)
	}))

	spa := func(w http.ResponseWriter, r *http.Request) {
		SetSessionCookie(w, token)
		serveIndex(w, r, root)
	}

	mux.HandleFunc("GET /{$}", spa)
	mux.HandleFunc("GET /review", spa)
	mux.HandleFunc("GET /entry", spa)
	mux.HandleFunc("GET /settings", spa)
	mux.HandleFunc("GET /overview", spa)
	mux.HandleFunc("GET /ingest", spa)
}

func serveIndex(w http.ResponseWriter, r *http.Request, root fs.FS) {
	raw, err := fs.ReadFile(root, "index.html")
	if err != nil {
		http.Error(w, "web ui not built; run: make web", http.StatusServiceUnavailable)
		return
	}
	locale := resolveLocale(r)
	boot, _ := json.Marshal(bootPayload{Locale: locale})
	inject := []byte("window.__ROCKY_BOOT__=" + string(boot) + ";")
	// Replace the empty boot assignment from the built index.html.
	out := bytes.Replace(raw, []byte("window.__ROCKY_BOOT__ = window.__ROCKY_BOOT__ || {};"), inject, 1)
	if bytes.Equal(out, raw) {
		// Vite may minify; try a looser match.
		out = bytes.Replace(raw, []byte("window.__ROCKY_BOOT__=window.__ROCKY_BOOT__||{}"), inject, 1)
	}
	if bytes.Equal(out, raw) {
		// Last resort: prepend boot before </head>.
		out = bytes.Replace(raw, []byte("</head>"), append(append([]byte("<script>"), inject...), []byte("</script></head>")...), 1)
	}
	// Ensure html lang attribute.
	out = bytes.Replace(out, []byte(`<html lang="en">`), []byte(`<html lang="`+locale+`">`), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(out)
}

func resolveLocale(r *http.Request) string {
	if q := strings.TrimSpace(r.URL.Query().Get("lang")); q != "" {
		return normalizeLocale(q)
	}
	if c, err := r.Cookie(LocaleCookie); err == nil && c.Value != "" {
		return normalizeLocale(c.Value)
	}
	if al := r.Header.Get("Accept-Language"); al != "" {
		return normalizeLocale(strings.Split(al, ",")[0])
	}
	return "en"
}

func normalizeLocale(raw string) string {
	t := strings.ToLower(strings.TrimSpace(raw))
	t = strings.ReplaceAll(t, "_", "-")
	if strings.HasPrefix(t, "zh") {
		return "zh-CN"
	}
	return "en"
}
