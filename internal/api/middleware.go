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

// API key authentication errors
var (
	ErrAuthInvalidToken = fmt.Errorf("invalid token")
	ErrAuthExpiredKey   = fmt.Errorf("expired API key")
	ErrAuthLegacyKey    = fmt.Errorf("legacy API key without user association")
	ErrAuthOwnerInvalid = fmt.Errorf("API key owner not found or inactive")
)

// AuthenticateAPIKey validates a Bearer token as an API key and returns the
// resolved Caller. This is the single source of truth for API key authentication
// used by both the REST API middleware and the MCP server.
//
// It enforces that all API keys must be associated with an active user — legacy
// keys (no UserID) are rejected.
func AuthenticateAPIKey(ctx context.Context, store storage.ExtendedStorage, token, ip, source string) (*service.Caller, error) {
	if store == nil {
		return nil, ErrAuthInvalidToken
	}

	hash := auth.HashToken(token)
	key, err := store.GetAPIKeyByKey(ctx, hash)
	if err != nil || subtle.ConstantTimeCompare([]byte(hash), []byte(key.Key)) != 1 {
		return nil, ErrAuthInvalidToken
	}

	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		log.Debug("Auth failed: expired API key", "key_name", key.Name)
		return nil, ErrAuthExpiredKey
	}

	// Update last used (async, don't block request)
	go func() {
		store.UpdateAPIKeyLastUsed(context.Background(), key.ID, time.Now())
	}()

	// API keys must be associated with a user to enforce RBAC
	if key.UserID == "" {
		log.Warn("Legacy API key rejected - no user association",
			"key_name", key.Name,
			"key_id", key.ID,
			"ip", ip,
		)
		return nil, ErrAuthLegacyKey
	}

	user, err := store.GetUser(ctx, key.UserID)
	if err != nil || !user.IsActive {
		log.Warn("API key owner not found or inactive", "key_name", key.Name, "user_id", key.UserID)
		return nil, ErrAuthOwnerInvalid
	}

	log.Trace("Auth successful (API key)", "key_name", key.Name, "source", source)
	return &service.Caller{
		Type:      service.CallerTypeUser,
		UserID:    user.ID,
		Username:  user.Username,
		IPAddress: ip,
		Source:    source,
	}, nil
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

		token := strings.TrimPrefix(authHeader, "Bearer ")
		caller, err := AuthenticateAPIKey(r.Context(), store, token, getClientIP(r, false), "api")
		if err != nil {
			code := "UNAUTHORIZED"
			if err == ErrAuthExpiredKey {
				code = "EXPIRED_KEY"
			} else if err == ErrAuthLegacyKey {
				code = "LEGACY_API_KEY_UNSUPPORTED"
			}
			log.Debug("Auth failed", "path", r.URL.Path, "error", err)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"Unauthorized","code":"%s"}`, code), http.StatusUnauthorized)
			return
		}

		r = r.WithContext(service.WithCaller(r.Context(), caller))
		next(w, r)
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

					// CSRF Protection for state-changing requests when using sessions (M-6)
					switch r.Method {
					case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
						// Safe methods don't require CSRF checks
					default:
						// Ensure it's a legitimate API request from the SPA
						if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
							log.Warn("CSRF blocked: Missing X-Requested-With header", "path", r.URL.Path, "username", session.Username)
							w.Header().Set("Content-Type", "application/json")
							http.Error(w, `{"error":"CSRF validation failed: missing custom header","code":"CSRF_FAILED"}`, http.StatusForbidden)
							return
						}
					}

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

		token := strings.TrimPrefix(authHeader, "Bearer ")
		caller, err := AuthenticateAPIKey(r.Context(), store, token, getClientIP(r, false), "api")
		if err != nil {
			code := "UNAUTHORIZED"
			if err == ErrAuthExpiredKey {
				code = "EXPIRED_KEY"
			} else if err == ErrAuthLegacyKey {
				code = "LEGACY_API_KEY_UNSUPPORTED"
			}
			log.Debug("Auth failed", "path", r.URL.Path, "error", err)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"Unauthorized","code":"%s"}`, code), http.StatusUnauthorized)
			return
		}

		r = r.WithContext(service.WithCaller(r.Context(), caller))
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
		// CSP: Removed 'unsafe-eval' and 'unsafe-inline' for scripts (M-7)
		// Now using @alpinejs/csp ensuring compliance
		// 'unsafe-inline' for styles remains due to occasional dynamic styles from JS that fail without it
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

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

		// Add rate limit headers
		remaining := limiter.GetRemaining(clientID)
		resetTime := limiter.GetResetTime(clientID)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.requests))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		next(w, r)
	}
}

// LogAuthWarning logs a warning when auth token is empty (open API mode)
func LogAuthWarning(token string) {
	if token == "" {
		log.Warn("API authentication is disabled - API is open to all requests")
	}
}
