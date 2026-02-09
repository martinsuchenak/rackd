package api

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/metrics"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// MaxRequestBodySize is the maximum allowed request body size (1MB)
const MaxRequestBodySize = 1 << 20

// AuthContext key for storing authenticated API key info
type contextKey string

const (
	APIKeyContextKey  contextKey = "apikey"
	SessionContextKey contextKey = "session"
	UserContextKey    contextKey = "user"

	sessionCookieName = "rackd_session"
)

// LoggingMiddleware logs all HTTP requests and records metrics
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Debug("HTTP request started",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
		)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Debug("HTTP request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		)

		// Record metrics
		metrics.Get().RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// resolveAPIKeyCaller builds a Caller for an authenticated API key.
// If the key has a UserID, it resolves the owner and returns a CallerTypeUser
// so that RBAC is enforced using the owner's roles. Legacy keys (no UserID)
// get CallerTypeAPIKey which bypasses RBAC.
func resolveAPIKeyCaller(store storage.ExtendedStorage, key *model.APIKey, ip, source string) *service.Caller {
	if key.UserID != "" {
		user, err := store.GetUser(key.UserID)
		if err == nil && user.IsActive {
			return &service.Caller{
				Type:      service.CallerTypeUser,
				UserID:    user.ID,
				Username:  user.Username,
				IPAddress: ip,
				Source:    source,
			}
		}
		log.Warn("API key owner not found or inactive", "key_name", key.Name, "user_id", key.UserID)
	}
	// Legacy key (no user association) — keep CallerTypeAPIKey
	return &service.Caller{
		Type:      service.CallerTypeAPIKey,
		UserID:    key.ID,
		Username:  key.Name,
		IPAddress: ip,
		Source:    source,
	}
}

// AuthMiddleware validates API keys
func AuthMiddleware(store storage.ExtendedStorage, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Debug("Auth failed: missing Bearer prefix", "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		providedToken := strings.TrimPrefix(authHeader, "Bearer ")

		// Try API key authentication
		if store != nil {
			key, err := store.GetAPIKeyByKey(providedToken)
			if err == nil && subtle.ConstantTimeCompare([]byte(providedToken), []byte(key.Key)) == 1 {
				// Check expiration
				if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
					log.Debug("Auth failed: expired API key", "path", r.URL.Path, "key_name", key.Name)
					w.Header().Set("Content-Type", "application/json")
					http.Error(w, `{"error":"Unauthorized","code":"EXPIRED_KEY"}`, http.StatusUnauthorized)
					return
				}

				// Update last used (async, don't block request)
				go func() {
					store.UpdateAPIKeyLastUsed(key.ID, time.Now())
				}()

				log.Trace("Auth successful (API key)", "path", r.URL.Path, "key_name", key.Name)
				caller := resolveAPIKeyCaller(store, key, getClientIP(r, false), "api")
				r = r.WithContext(service.WithCaller(r.Context(), caller))
				next(w, r)
				return
			}
		}

		log.Debug("Auth failed: invalid token", "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
	}
}

// AuthMiddlewareWithSessions validates API keys and sessions
func AuthMiddlewareWithSessions(store storage.ExtendedStorage, sessionManager *auth.SessionManager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try session authentication via cookie first
		if sessionManager != nil {
			if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
				session, err := sessionManager.GetSession(cookie.Value)
				if err == nil {
					sessionManager.RefreshSession(cookie.Value)
					log.Trace("Auth successful (session cookie)", "path", r.URL.Path, "username", session.Username)
					r = r.WithContext(context.WithValue(r.Context(), SessionContextKey, session))
					caller := &service.Caller{
						Type:      service.CallerTypeUser,
						UserID:    session.UserID,
						Username:  session.Username,
						IPAddress: getClientIP(r, false),
						Source:    "api",
					}
					r = r.WithContext(service.WithCaller(r.Context(), caller))
					next(w, r)
					return
				}
			}
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Debug("Auth failed: no session cookie or Bearer token", "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		providedToken := strings.TrimPrefix(authHeader, "Bearer ")

		// Try API key authentication
		if store != nil {
			key, err := store.GetAPIKeyByKey(providedToken)
			if err == nil && subtle.ConstantTimeCompare([]byte(providedToken), []byte(key.Key)) == 1 {
				// Check expiration
				if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
					log.Debug("Auth failed: expired API key", "path", r.URL.Path, "key_name", key.Name)
					w.Header().Set("Content-Type", "application/json")
					http.Error(w, `{"error":"Unauthorized","code":"EXPIRED_KEY"}`, http.StatusUnauthorized)
					return
				}

				// Update last used (async, don't block request)
				go func() {
					store.UpdateAPIKeyLastUsed(key.ID, time.Now())
				}()

				log.Trace("Auth successful (API key)", "path", r.URL.Path, "key_name", key.Name)
				caller := resolveAPIKeyCaller(store, key, getClientIP(r, false), "api")
				r = r.WithContext(service.WithCaller(r.Context(), caller))
				next(w, r)
				return
			}
		}

		log.Debug("Auth failed: invalid token", "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
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
		// 'unsafe-inline' for scripts is required for Alpine.js inline event handlers (@click, etc.).
		// Consider using Alpine.js CSP build (@alpinejs/csp) to remove these requirements.
		// 'unsafe-inline' for styles is required for Tailwind's dynamic classes.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

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

// LoginRateLimitMiddleware wraps a handler with a strict per-IP rate limiter
// for the login endpoint. Unlike the global rate limiter, this does NOT bypass localhost.
func LoginRateLimitMiddleware(limiter *RateLimiter, trustProxy bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := getClientIP(r, trustProxy)

		if !limiter.Allow(clientID) {
			resetTime := limiter.GetResetTime(clientID)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.requests))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
			w.Header().Set("Retry-After", resetTime.Format(time.RFC3339))

			log.Warn("Login rate limit exceeded", "client", clientID)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Too many login attempts. Please try again later.","code":"LOGIN_RATE_LIMIT_EXCEEDED"}`, http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// LogAuthWarning logs a warning when auth token is empty (open API mode)
func LogAuthWarning(token string) {
	if token == "" {
		log.Warn("API authentication is disabled - API is open to all requests")
	}
}
