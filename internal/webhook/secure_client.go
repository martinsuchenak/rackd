package webhook

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// allowPrivateRanges permits dialing RFC1918/CGNAT/ULA destinations for
// operators who intentionally point integrations at internal services
// (e.g. a Technitium DNS server on 10.0.0.0/8). Loopback, link-local
// (cloud metadata), unspecified and multicast addresses are always blocked.
// Set via SetPrivateRangePolicy during process startup from configuration.
var allowPrivateRanges atomic.Bool

// SetPrivateRangePolicy configures whether private-range destinations are
// permitted for outbound integration requests. It is intended to be called
// once during process startup before any request is issued.
func SetPrivateRangePolicy(allow bool) {
	allowPrivateRanges.Store(allow)
}

// isCGNAT reports whether ip is inside the RFC 6598 carrier-grade NAT range
// 100.64.0.0/10, which net.IP.IsPrivate does not cover.
func isCGNAT(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127
}

// blockedIPReason returns a human-readable reason when the IP must not be
// dialed by server-issued outbound requests, or "" when it is allowed.
// Loopback, unspecified, link-local (covers cloud metadata 169.254.0.0/16),
// multicast and (unless explicitly allowed) RFC1918, ULA and CGNAT ranges
// are blocked.
func blockedIPReason(ip net.IP) string {
	switch {
	case ip.IsLoopback():
		return "loopback address"
	case ip.IsUnspecified():
		return "unspecified address"
	case ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast():
		return "link-local address"
	case ip.IsMulticast():
		return "multicast address"
	}
	if !allowPrivateRanges.Load() {
		if ip.IsPrivate() {
			return "private address"
		}
		if isCGNAT(ip) {
			return "private address"
		}
	}
	return ""
}

// ValidateOutboundHost reports whether host (an URL host, i.e. a literal IP
// or a hostname) resolves only to destinations that outbound integration
// requests may reach. Hostnames are resolved so that names pointing at
// private infrastructure are rejected at validation time as well.
func ValidateOutboundHost(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if reason := blockedIPReason(ip); reason != "" {
			return fmt.Errorf("SSRF prevention: %s is not allowed (%s)", host, reason)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("SSRF prevention: failed to resolve host %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("SSRF prevention: host %s has no addresses", host)
	}
	for _, ip := range ips {
		if reason := blockedIPReason(ip); reason != "" {
			return fmt.Errorf("SSRF prevention: %s resolves to %s (%s)", host, ip.String(), reason)
		}
	}
	return nil
}

// SafeDialContext is a DialContext function that prevents connections to
// loopback and link-local addresses to mitigate SSRF vulnerabilities.
func SafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	var lastErr error
	for _, ip := range ips {
		// Block loopback, unspecified, link-local (cloud metadata),
		// multicast and — unless explicitly allowed by policy — private
		// ranges (RFC1918, ULA, CGNAT).
		if reason := blockedIPReason(ip); reason != "" {
			return nil, fmt.Errorf("SSRF prevention: connection to %s blocked (restricted IP %s: %s)", host, ip.String(), reason)
		}

		// Prevent DNS rebinding by dialing the exact IP we just verified
		addrWithIP := net.JoinHostPort(ip.String(), port)
		conn, err := dialer.DialContext(ctx, network, addrWithIP)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to connect to %s", addr)
}

// CheckRedirect policy that re-validates redirect targets: a public webhook
// URL must not be able to 302 into loopback/metadata addresses, which would
// otherwise bypass SafeDialContext's original-URL checks via a redirect chain.
func safeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	host := req.URL.Hostname()
	if host == "" {
		return fmt.Errorf("SSRF prevention: redirect with empty host blocked")
	}
	// Resolve and validate every redirect target with the same rules as
	// SafeDialContext (the dialer re-checks at connect time too, but rejecting
	// here avoids leaking the request to DNS-rebinding race windows).
	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if reason := blockedIPReason(ip); reason != "" {
			return fmt.Errorf("SSRF prevention: redirect to %s blocked (restricted IP %s: %s)", host, ip.String(), reason)
		}
	}
	return nil
}

// NewSecureHTTPClient returns an http.Client that enforces SSRF protections
func NewSecureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: safeCheckRedirect,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           SafeDialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
