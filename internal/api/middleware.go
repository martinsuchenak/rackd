package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/internal/log"
)

// MaxRequestBodySize is the maximum allowed request body size (1MB)
const MaxRequestBodySize = 1 << 20

// AuthMiddleware validates bearer tokens using timing-safe comparison
func AuthMiddleware(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		providedToken := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(providedToken), []byte(token)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// CSP: 'unsafe-eval' required for Alpine.js x-data expressions.
		// Consider using Alpine.js CSP build (@alpinejs/csp) to remove this requirement.
		// 'unsafe-inline' for styles is required for Tailwind's dynamic classes.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		// HSTS only for TLS connections
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// LimitBody wraps a handler to limit request body size
func LimitBody(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
		next(w, r)
	}
}

// LogAuthWarning logs a warning when auth token is empty (open API mode)
func LogAuthWarning(token string) {
	if token == "" {
		log.Warn("API authentication is disabled - API is open to all requests")
	}
}
