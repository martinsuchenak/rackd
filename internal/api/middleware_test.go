package api

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	log.Init("console", "error", io.Discard)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create API key
	key := &model.APIKey{
		Name: "test-key",
		Key:  "secret-token",
	}
	if err := store.CreateAPIKey(key); err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	called := false
	handler := AuthMiddleware(store, func(w http.ResponseWriter, r *http.Request) {
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
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	called := false
	handler := AuthMiddleware(store, func(w http.ResponseWriter, r *http.Request) {
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
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	called := false
	handler := AuthMiddleware(store, func(w http.ResponseWriter, r *http.Request) {
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
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	called := false
	handler := AuthMiddleware(store, func(w http.ResponseWriter, r *http.Request) {
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

func TestAuthMiddleware_ExpiredKey(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create expired API key
	expired := time.Now().Add(-1 * time.Hour)
	key := &model.APIKey{
		Name:      "expired-key",
		Key:       "expired-token",
		ExpiresAt: &expired,
	}
	if err := store.CreateAPIKey(key); err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	called := false
	handler := AuthMiddleware(store, func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	handler(w, r)

	if called {
		t.Error("handler should not have been called for expired key")
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
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Permissions-Policy":      "geolocation=(), microphone=(), camera=()",
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
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
