package api

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/log"
)

func init() {
	log.Init("console", "error", io.Discard)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	called := false
	handler := AuthMiddleware("secret-token", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()

	handler(w, r)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	called := false
	handler := AuthMiddleware("secret-token", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()

	handler(w, r)

	if called {
		t.Error("handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_MissingBearer(t *testing.T) {
	called := false
	handler := AuthMiddleware("secret-token", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()

	handler(w, r)

	if called {
		t.Error("handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	called := false
	handler := AuthMiddleware("secret-token", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	if called {
		t.Error("handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestSecurityHeaders_HTTP(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"X-XSS-Protection":        "1; mode=block",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
	}

	for header, expected := range expectedHeaders {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("expected %s: %s, got %s", header, expected, got)
		}
	}

	// HSTS should NOT be set for HTTP
	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS should not be set for HTTP, got %s", hsts)
	}
}

func TestSecurityHeaders_HTTPS(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/", nil)
	r.TLS = &tls.ConnectionState{} // Simulate TLS connection
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	// HSTS should be set for HTTPS
	expected := "max-age=31536000; includeSubDomains"
	if got := w.Header().Get("Strict-Transport-Security"); got != expected {
		t.Errorf("expected HSTS: %s, got %s", expected, got)
	}
}

func TestLogAuthWarning(t *testing.T) {
	// Just ensure it doesn't panic
	LogAuthWarning("")
	LogAuthWarning("some-token")
}

func TestLimitBody(t *testing.T) {
	handler := LimitBody(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Test with small body - should succeed
	smallBody := make([]byte, 1024)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(smallBody))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for small body, got %d", http.StatusOK, w.Code)
	}

	// Test with body exceeding limit - should fail
	largeBody := make([]byte, MaxRequestBodySize+1)
	r = httptest.NewRequest("POST", "/", bytes.NewReader(largeBody))
	w = httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d for large body, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}
