package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Second)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		if !limiter.Allow("client1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if limiter.Allow("client1") {
		t.Error("4th request should be blocked")
	}

	// Different client should not be affected
	if !limiter.Allow("client2") {
		t.Error("Different client should be allowed")
	}

	// Wait for window to reset
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow("client1") {
		t.Error("Request after window reset should be allowed")
	}
}

func TestRateLimiterGetRemaining(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Second)

	if remaining := limiter.GetRemaining("client1"); remaining != 5 {
		t.Errorf("Expected 5 remaining, got %d", remaining)
	}

	limiter.Allow("client1")
	limiter.Allow("client1")

	if remaining := limiter.GetRemaining("client1"); remaining != 3 {
		t.Errorf("Expected 3 remaining, got %d", remaining)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Second)
	middleware := RateLimitMiddleware(limiter, false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}

	// Check rate limit headers
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Missing X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("Expected X-RateLimit-Remaining: 0, got %s", w.Header().Get("X-RateLimit-Remaining"))
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Missing X-RateLimit-Reset header")
	}
}

func TestRateLimitMiddlewareLocalhostBypass(t *testing.T) {
	limiter := NewRateLimiter(1, 1*time.Second)
	middleware := RateLimitMiddleware(limiter, false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Localhost should bypass rate limiting
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Localhost request %d should not be rate limited, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddlewareAPIKey(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Second)
	middleware := RateLimitMiddleware(limiter, false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Requests with API key should be rate limited by key
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		req.Header.Set("Authorization", "Bearer test-key-123")
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request with same key should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer test-key-123")
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}

	// Different key should not be affected
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer different-key")
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Different key should be allowed, got %d", w.Code)
	}
}

func TestGetClientIP_TrustProxy(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expected   string
	}{
		{"RemoteAddr", "192.168.1.1:1234", "", "", "192.168.1.1"},
		{"X-Forwarded-For", "192.168.1.1:1234", "10.0.0.1, 10.0.0.2", "", "10.0.0.1"},
		{"X-Real-IP", "192.168.1.1:1234", "", "10.0.0.1", "10.0.0.1"},
		{"IPv6", "[::1]:1234", "", "", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			ip := getClientIP(req, true)
			if ip != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestGetClientIP_NoTrustProxy(t *testing.T) {
	// When trustProxy is false, X-Forwarded-For and X-Real-IP should be ignored
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Real-IP", "10.0.0.2")

	ip := getClientIP(req, false)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1 (RemoteAddr), got %s (proxy headers should be ignored)", ip)
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"127.0.0.1:1234", true},
		{"[::1]:1234", true},
		{"localhost:1234", true},
		{"192.168.1.1:1234", false},
		{"10.0.0.1:1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := isLocalhost(tt.addr)
			if result != tt.expected {
				t.Errorf("isLocalhost(%s) = %v, expected %v", tt.addr, result, tt.expected)
			}
		})
	}
}
