package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed assets/*
var assets embed.FS

// RegisterRoutes serves the embedded UI assets with SPA fallback
func RegisterRoutes(mux *http.ServeMux) {
	sub, _ := fs.Sub(assets, "assets")

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || !hasExtension(r.URL.Path) {
			path = "index.html"
		}

		data, err := fs.ReadFile(sub, path)
		if err != nil {
			// File not found - serve index.html for SPA
			data, _ = fs.ReadFile(sub, "index.html")
			path = "index.html"
		}

		w.Header().Set("Content-Type", contentType(path))
		w.Write(data)
	})
}

func hasExtension(path string) bool {
	lastSlash := strings.LastIndex(path, "/")
	lastDot := strings.LastIndex(path, ".")
	return lastDot > lastSlash
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
