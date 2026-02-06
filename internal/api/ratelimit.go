package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
)

// RateLimiter tracks request rates per client
type RateLimiter struct {
	mu       sync.RWMutex
	clients  map[string]*clientBucket
	requests int
	window   time.Duration
	cleanup  time.Duration
}

type clientBucket struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a rate limiter with specified requests per window
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*clientBucket),
		requests: requests,
		window:   window,
		cleanup:  window * 2,
	}
	go rl.cleanupLoop()
	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.RLock()
	bucket, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		bucket = &clientBucket{
			tokens:    rl.requests,
			lastReset: time.Now(),
		}
		rl.mu.Lock()
		rl.clients[clientID] = bucket
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Reset bucket if window expired
	if time.Since(bucket.lastReset) > rl.window {
		bucket.tokens = rl.requests
		bucket.lastReset = time.Now()
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// GetRemaining returns remaining tokens for a client
func (rl *RateLimiter) GetRemaining(clientID string) int {
	rl.mu.RLock()
	bucket, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		return rl.requests
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if time.Since(bucket.lastReset) > rl.window {
		return rl.requests
	}

	return bucket.tokens
}

// GetResetTime returns when the bucket will reset
func (rl *RateLimiter) GetResetTime(clientID string) time.Time {
	rl.mu.RLock()
	bucket, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		return time.Now().Add(rl.window)
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	return bucket.lastReset.Add(rl.window)
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, bucket := range rl.clients {
			bucket.mu.Lock()
			if now.Sub(bucket.lastReset) > rl.cleanup {
				delete(rl.clients, id)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware applies rate limiting to requests
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass for localhost
			if isLocalhost(r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}

			// Use API key as client ID if present, otherwise use IP
			clientID := getClientIP(r)
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				clientID = strings.TrimPrefix(auth, "Bearer ")
			}

			if !limiter.Allow(clientID) {
				resetTime := limiter.GetResetTime(clientID)
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.requests))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
				w.Header().Set("Retry-After", resetTime.Format(time.RFC3339))
				
				log.Debug("Rate limit exceeded", "client", clientID, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Rate limit exceeded","code":"RATE_LIMIT_EXCEEDED"}`, http.StatusTooManyRequests)
				return
			}

			// Add rate limit headers
			remaining := limiter.GetRemaining(clientID)
			resetTime := limiter.GetResetTime(clientID)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use RemoteAddr - use net.SplitHostPort to handle both IPv4 and IPv6
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If splitting fails, return as-is (may not have port)
		return r.RemoteAddr
	}
	return host
}

func isLocalhost(addr string) bool {
	ip := addr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	ip = strings.Trim(ip, "[]")
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}
