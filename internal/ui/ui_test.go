package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux)

	tests := []struct {
		path     string
		wantCode int
		contains string
	}{
		{"/", http.StatusOK, "<!DOCTYPE html>"},
		{"/index.html", http.StatusOK, "<!DOCTYPE html>"},
		{"/app.js", http.StatusOK, "placeholder"},
		{"/output.css", http.StatusOK, "font-family"},
		{"/devices", http.StatusOK, "<!DOCTYPE html>"}, // SPA fallback
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
