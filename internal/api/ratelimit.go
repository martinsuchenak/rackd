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

// TrustedProxyResolver decides whether proxy-supplied client-IP headers may
// be trusted for a given request, based on the immediate peer address
// (RemoteAddr) matching a configured trusted-proxy CIDR/network.
type TrustedProxyResolver struct {
	trustProxy  bool
	trustedNets []*net.IPNet
}

// NewTrustedProxyResolver parses a comma-separated list of IPs and CIDR
// ranges. When trustProxy is false or no ranges are configured, the resolver
// never trusts proxy headers (fail-closed).
func NewTrustedProxyResolver(trustProxy bool, trustedProxies string) *TrustedProxyResolver {
	resolver := &TrustedProxyResolver{trustProxy: trustProxy}
	if !trustProxy {
		return resolver
	}
	for _, entry := range strings.Split(trustedProxies, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			} else {
				log.Warn("Ignoring invalid TRUSTED_PROXIES entry", "entry", entry)
				continue
			}
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			log.Warn("Ignoring invalid TRUSTED_PROXIES entry", "entry", entry)
			continue
		}
		resolver.trustedNets = append(resolver.trustedNets, network)
	}
	if len(resolver.trustedNets) == 0 {
		log.Warn("TRUST_PROXY=true but no valid TRUSTED_PROXIES configured; proxy headers will be ignored")
	}
	return resolver
}

// Trusted reports whether the request's immediate peer is a configured
// trusted proxy. Proxy headers from any other source are attacker-controlled
// and must not influence rate-limit bucket selection.
func (t *TrustedProxyResolver) Trusted(r *http.Request) bool {
	if t == nil || !t.trustProxy || len(t.trustedNets) == 0 {
		return false
	}
	peer := remoteIP(r)
	ip := net.ParseIP(peer)
	if ip == nil {
		return false
	}
	for _, network := range t.trustedNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

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
func RateLimitMiddleware(limiter *RateLimiter, proxyResolver *TrustedProxyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Key by client IP only. Keying on the raw Authorization header
			// let anonymous clients rotate fake Bearer tokens to get fresh
			// buckets (bypass) and grow the bucket map without bound.
			clientID := getClientIP(r, proxyResolver)

			if !limiter.Allow(clientID) {
				resetTime := limiter.GetResetTime(clientID)
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.requests))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
				w.Header().Set("Retry-After", resetTime.Format(time.RFC3339))

				log.Debug("Rate limit exceeded", "client", clientID, "path", r.URL.Path)
				writeAuthError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded")
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

func getClientIP(r *http.Request, proxyResolver *TrustedProxyResolver) string {
	if proxyResolver.Trusted(r) {
		// Only honor proxy headers when the immediate peer is a configured
		// trusted proxy. Use the RIGHTMOST XFF entry: it is the address
		// appended by our own trusted reverse proxy. The leftmost entries are
		// client-supplied and can be rotated to obtain fresh rate-limit
		// buckets.
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			return strings.TrimSpace(ips[len(ips)-1])
		}

		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}

	// Use RemoteAddr - use net.SplitHostPort to handle both IPv4 and IPv6
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If splitting fails, return as-is (may not have port)
		return r.RemoteAddr
	}
	return host
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
