package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/log"
)

func init() {
	log.Init("console", "error", io.Discard)
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(nil, nil)
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
}

func TestWriteJSON(t *testing.T) {
	h := NewHandler(nil, nil)
	w := httptest.NewRecorder()

	h.writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	if w.Body.String() != `{"key":"value"}`+"\n" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestWriteError(t *testing.T) {
	h := NewHandler(nil, nil)
	w := httptest.NewRecorder()

	h.writeError(w, http.StatusBadRequest, "TEST_ERROR", "Test message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestInternalError(t *testing.T) {
	h := NewHandler(nil, nil)
	w := httptest.NewRecorder()

	h.internalError(w, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestParseArrayParam(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		param    string
		expected []string
	}{
		{"empty", "", "tags", nil},
		{"single", "tags=foo", "tags", []string{"foo"}},
		{"multiple", "tags=foo&tags=bar", "tags", []string{"foo", "bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/?"+tt.query, nil)
			result := parseArrayParam(r, tt.param)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		param    string
		def      int
		expected int
	}{
		{"empty", "", "limit", 100, 100},
		{"valid", "limit=50", "limit", 100, 50},
		{"invalid", "limit=abc", "limit", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/?"+tt.query, nil)
			result := parseIntParam(r, tt.param, tt.def)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestWithAuth(t *testing.T) {
	opt := WithAuth()
	cfg := &handlerConfig{}
	opt(cfg)
	if !cfg.requireAuth {
		t.Error("expected requireAuth to be true")
	}
}

func TestRegisterRoutes(t *testing.T) {
	h := NewHandler(nil, nil)

	// Test without auth
	mux1 := http.NewServeMux()
	h.RegisterRoutes(mux1)

	// Test with auth - use separate mux
	mux2 := http.NewServeMux()
	h.RegisterRoutes(mux2, WithAuth())
}
