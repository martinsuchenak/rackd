package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed assets/*
var assets embed.FS

// Embedded returns the web UI built into this binary.
func Embedded() fs.FS {
	sub, _ := fs.Sub(assets, "assets")
	return sub
}

// IndexHTML returns the built-in index.html content for SPA routing
func IndexHTML() []byte {
	data, _ := fs.ReadFile(Embedded(), "index.html")
	return data
}

// Site serves a web UI's files, with index.html for any other route.
type Site struct {
	fsys  fs.FS
	index []byte
}

// New returns a Site for fsys, or for the built-in UI when fsys is nil. fsys
// must hold index.html at its root.
func New(fsys fs.FS) (*Site, error) {
	if fsys == nil {
		fsys = Embedded()
	}
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return nil, fmt.Errorf("web UI: %w", err)
	}
	return &Site{fsys: fsys, index: index}, nil
}

// IndexHTML returns the site's index.html.
func (s *Site) IndexHTML() []byte { return s.index }

// RegisterRoutes serves the site's assets with SPA fallback
func (s *Site) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		var data []byte
		if path != "" && hasExtension(r.URL.Path) {
			data, _ = fs.ReadFile(s.fsys, path)
		}
		if data == nil {
			// Not a file - serve index.html for SPA
			data, path = s.index, "index.html"
		}

		w.Header().Set("Content-Type", contentType(path))
		_, _ = w.Write(data)
	})
}

// RegisterRoutes serves the built-in UI.
func RegisterRoutes(mux *http.ServeMux) {
	site, _ := New(nil)
	site.RegisterRoutes(mux)
}

// ExtensionPrefix is where an extension's UI files are served.
func ExtensionPrefix(name string) string { return "/ext/" + name + "/" }

// RegisterExtensionAssets serves an extension's UI files under
// ExtensionPrefix(name). Unknown files and directories are 404s, so a typo in
// a script path fails visibly instead of loading index.html as JavaScript.
func RegisterExtensionAssets(mux *http.ServeMux, name string, fsys fs.FS) {
	prefix := ExtensionPrefix(name)
	mux.HandleFunc("GET "+prefix, func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, prefix)
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType(path))
		_, _ = w.Write(data)
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
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".mjs"):
		return "application/javascript"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".json"), strings.HasSuffix(path, ".map"):
		return "application/json"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}
