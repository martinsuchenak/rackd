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
	middleware := RateLimitMiddleware(limiter, NewTrustedProxyResolver(false, ""))

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

func TestRateLimitMiddlewareNoLocalhostBypass(t *testing.T) {
	limiter := NewRateLimiter(1, 1*time.Second)
	middleware := RateLimitMiddleware(limiter, NewTrustedProxyResolver(false, ""))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// First request from localhost should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("First localhost request should succeed, got %d", w.Code)
	}

	// Second request from localhost should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Second localhost request should be rate limited, got %d", w.Code)
	}
}

func TestRateLimitMiddlewareAPIKey(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Second)
	middleware := RateLimitMiddleware(limiter, NewTrustedProxyResolver(false, ""))

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

	// Rotating a different fake Bearer token must NOT bypass the limit
	// (the limiter keys by client IP, not the raw Authorization header).
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer different-key")
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Bearer rotation should not bypass rate limit, got %d", w.Code)
	}

	// A different source IP is a different bucket
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	req.Header.Set("Authorization", "Bearer different-key")
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Different IP should be allowed, got %d", w.Code)
	}
}

func TestGetClientIP_TrustProxy(t *testing.T) {
	// The immediate peer (RemoteAddr) is a configured trusted proxy, so its
	// XFF/X-Real-IP headers may be honored.
	resolver := NewTrustedProxyResolver(true, "10.0.0.2")
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expected   string
	}{
		// Peers other than the trusted proxy keep their RemoteAddr key.
		{"RemoteAddr untrusted", "192.168.1.1:1234", "", "", "192.168.1.1"},
		{"IPv6", "[::1]:1234", "", "", "::1"},
		// Requests arriving from the trusted proxy itself.
		{"RemoteAddr trusted no headers", "10.0.0.2:1234", "", "", "10.0.0.2"},
		// The RIGHTMOST XFF entry is the one appended by our own trusted proxy;
		// leftmost entries are client-controlled and spoofable.
		{"X-Forwarded-For", "10.0.0.2:1234", "10.0.0.1, 10.0.0.2", "", "10.0.0.2"},
		{"X-Real-IP", "10.0.0.2:1234", "", "10.0.0.1", "10.0.0.1"},
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

			ip := getClientIP(req, resolver)
			if ip != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestGetClientIP_NoTrustProxy(t *testing.T) {
	// When trustProxy is false, X-Forwarded-For and X-Real-IP should be ignored
	resolver := NewTrustedProxyResolver(false, "10.0.0.0/8")
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Real-IP", "10.0.0.2")

	ip := getClientIP(req, resolver)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1 (RemoteAddr), got %s (proxy headers should be ignored)", ip)
	}
}

func TestGetClientIP_UntrustedPeerIgnoresProxyHeaders(t *testing.T) {
	// TRUST_PROXY=true with a trusted proxy configured, but the immediate
	// peer is NOT in the trusted list: spoofed headers must be ignored and
	// the key must fall back to RemoteAddr. This is the regression test for
	// the rate-limit bucket rotation bypass via X-Forwarded-For / X-Real-IP.
	resolver := NewTrustedProxyResolver(true, "10.9.9.0/24")

	for _, header := range []struct {
		name   string
		key    string
		values []string
	}{
		{"X-Real-IP", "X-Real-IP", []string{"1.2.3.4", "5.6.7.8"}},
		{"XFF single", "X-Forwarded-For", []string{"1.2.3.4", "5.6.7.8"}},
		{"XFF rightmost", "X-Forwarded-For", []string{"198.51.100.1, 203.0.113.1", "198.51.100.2, 203.0.113.2"}},
	} {
		t.Run(header.name, func(t *testing.T) {
			for _, value := range header.values {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "203.0.113.99:4444" // untrusted peer
				req.Header.Set(header.key, value)
				if ip := getClientIP(req, resolver); ip != "203.0.113.99" {
					t.Errorf("spoofed %s=%q from untrusted peer changed key to %q; want RemoteAddr", header.key, value, ip)
				}
			}
		})
	}
}

func TestGetClientIP_TrustedProxyNoRangesFailClosed(t *testing.T) {
	// TRUST_PROXY=true but TRUSTED_PROXIES unset: fail closed — proxy
	// headers from any peer are ignored (they cannot be authenticated).
	resolver := NewTrustedProxyResolver(true, "")
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	if ip := getClientIP(req, resolver); ip != "192.168.1.1" {
		t.Errorf("Expected RemoteAddr fallback, got %s", ip)
	}
}

func TestGetClientIP_TrustedProxyHonorsForwardedFor(t *testing.T) {
	// Peer is the configured proxy: rightmost XFF (appended by the proxy)
	// is used, so distinct real clients get distinct buckets.
	resolver := NewTrustedProxyResolver(true, "10.0.0.2")
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.2:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 203.0.113.20")
	if ip := getClientIP(req, resolver); ip != "203.0.113.20" {
		t.Errorf("Expected rightmost XFF 203.0.113.20, got %s", ip)
	}
	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "203.0.113.30")
	if ip := getClientIP(req, resolver); ip != "203.0.113.30" {
		t.Errorf("Expected X-Real-IP 203.0.113.30, got %s", ip)
	}
}
