package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRegisterRoutes(t *testing.T) {
	site, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	site.RegisterRoutes(mux)

	tests := []struct {
		path     string
		wantCode int
		contains string
	}{
		{"/", http.StatusOK, "<!DOCTYPE html>"},
		{"/index.html", http.StatusOK, "<!DOCTYPE html>"},
		{"/app.js", http.StatusOK, "placeholder"},
		{"/output.css", http.StatusOK, "font-family"},
		{"/devices", http.StatusOK, "<!DOCTYPE html>"},      // SPA fallback
		{"/networks/123", http.StatusOK, "<!DOCTYPE html>"}, // SPA fallback
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("path %s: got status %d, want %d", tt.path, w.Code, tt.wantCode)
			}
			if !strings.Contains(w.Body.String(), tt.contains) {
				t.Errorf("path %s: body missing %q", tt.path, tt.contains)
			}
		})
	}
}

func TestHasExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/", false},
		{"/devices", false},
		{"/app.js", true},
		{"/output.css", true},
		{"/path/to/file.html", true},
		{"/path.with.dots/noext", false},
	}

	for _, tt := range tests {
		if got := hasExtension(tt.path); got != tt.want {
			t.Errorf("hasExtension(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSiteServesAReplacementUI(t *testing.T) {
	if _, err := New(fstest.MapFS{"app.js": {}}); err == nil {
		t.Error("New accepted a UI without index.html")
	}
	site, err := New(fstest.MapFS{
		"index.html": {Data: []byte("custom index")},
		"app.js":     {Data: []byte("custom js")},
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	site.RegisterRoutes(mux)
	for path, want := range map[string]string{"/": "custom index", "/devices": "custom index", "/app.js": "custom js", "/missing.css": "custom index"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Body.String() != want {
			t.Errorf("GET %s = %q, want %q", path, w.Body.String(), want)
		}
	}
}

func TestRegisterExtensionAssets(t *testing.T) {
	mux := http.NewServeMux()
	RegisterExtensionAssets(mux, "notes", fstest.MapFS{
		"ui/index.js":   {Data: []byte("export {};")},
		"ui/notes.css":  {Data: []byte("p{}")},
		"ui/chunk#1.js": {Data: []byte("export {};")},
	})
	tests := []struct {
		path        string
		wantCode    int
		contentType string
	}{
		{"/ext/notes/ui/index.js", http.StatusOK, "application/javascript"},
		{"/ext/notes/ui/notes.css", http.StatusOK, "text/css"},
		{ExtensionURL("notes", "ui/chunk#1.js"), http.StatusOK, "application/javascript"},
		// Not index.html: a wrong script path must fail, not load HTML.
		{"/ext/notes/ui/missing.js", http.StatusNotFound, ""},
		{"/ext/notes/ui", http.StatusNotFound, ""},
		{"/ext/notes/", http.StatusNotFound, ""},
		{"/ext/other/ui/index.js", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		if w.Code != tt.wantCode {
			t.Errorf("GET %s = %d, want %d", tt.path, w.Code, tt.wantCode)
		}
		if tt.contentType != "" && w.Header().Get("Content-Type") != tt.contentType {
			t.Errorf("GET %s Content-Type = %q", tt.path, w.Header().Get("Content-Type"))
		}
	}
}
