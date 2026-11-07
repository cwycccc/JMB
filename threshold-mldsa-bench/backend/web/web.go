// Package web serves only the embedded public dashboard assets.
package web

import (
	"embed"
	"net/http"
)

//go:embed index.html style.css app.js
var assets embed.FS

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(405)
			return
		}
		name := r.URL.Path
		switch name {
		case "/":
			name = "index.html"
		case "/style.css":
			name = "style.css"
		case "/app.js":
			name = "app.js"
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		switch name {
		case "index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case "style.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		default:
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}
		data, _ := assets.ReadFile(name)
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	})
}
