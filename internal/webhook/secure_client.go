package webhook

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// allowPrivateRanges permits dialing every RFC1918/CGNAT/ULA destination
// (SSRF_ALLOW_PRIVATE_TARGETS); allowedPrivateTargets permits only listed
// ones, for operators who point integrations at internal services (e.g. a
// Technitium DNS server on 10.0.0.0/8). Loopback, link-local
// (cloud metadata), unspecified and multicast addresses are always blocked.
// Set via SetPrivateRangePolicy during process startup from configuration.
var allowPrivateRanges atomic.Bool

// SetPrivateRangePolicy configures whether private-range destinations are
// permitted for outbound integration requests. It is intended to be called
// once during process startup before any request is issued.
func SetPrivateRangePolicy(allow bool) {
	allowPrivateRanges.Store(allow)
}

// allowedPrivateTargets are the private ranges outbound requests may reach
// when private ranges as a whole are not allowed (SSRF_ALLOWED_TARGETS).
var allowedPrivateTargets atomic.Pointer[[]*net.IPNet]

// privateRanges are the ranges blocked unless allowed: RFC 1918, CGNAT and
// IPv6 unique-local.
var privateRanges = []*net.IPNet{
	mustCIDR("10.0.0.0/8"), mustCIDR("172.16.0.0/12"), mustCIDR("192.168.0.0/16"),
	mustCIDR("100.64.0.0/10"), mustCIDR("fc00::/7"),
}

// ParseAllowedTargets parses SSRF_ALLOWED_TARGETS entries: IP addresses or
// CIDR ranges, each inside a private range. Only the private-range block can
// be lifted this way; loopback, link-local (cloud metadata) and the other
// always-blocked ranges stay blocked, and a public range needs no entry.
func ParseAllowedTargets(entries []string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		var n *net.IPNet
		if ip := net.ParseIP(entry); ip != nil {
			bits := 128
			if ip4 := ip.To4(); ip4 != nil {
				ip, bits = ip4, 32
			}
			n = &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
		} else {
			var err error
			if _, n, err = net.ParseCIDR(entry); err != nil {
				return nil, fmt.Errorf("%q is not an IP address or CIDR range", entry)
			}
		}
		if !withinPrivateRange(n) {
			return nil, fmt.Errorf("%q is not inside a private range (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 100.64.0.0/10, fc00::/7)", entry)
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func withinPrivateRange(n *net.IPNet) bool {
	ones, bits := n.Mask.Size()
	for _, r := range privateRanges {
		rOnes, rBits := r.Mask.Size()
		if bits == rBits && ones >= rOnes && r.Contains(n.IP) {
			return true
		}
	}
	return false
}

// SetAllowedPrivateTargets sets the private ranges outbound requests may
// reach (from ParseAllowedTargets). Like SetPrivateRangePolicy, it is meant
// to be called once at startup.
func SetAllowedPrivateTargets(nets []*net.IPNet) {
	allowedPrivateTargets.Store(&nets)
}

func privateTargetAllowed(ip net.IP) bool {
	if allowPrivateRanges.Load() {
		return true
	}
	if nets := allowedPrivateTargets.Load(); nets != nil {
		for _, n := range *nets {
			if n.Contains(ip) {
				return true
			}
		}
	}
	return false
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
	// An IPv6 address that carries an IPv4 one (NAT64, 6to4, Teredo) can
	// reach that IPv4 address through a gateway, so it gets the IPv4 rules.
	if embedded := embeddedIPv4(ip); embedded != nil {
		if reason := blockedIPReason(embedded); reason != "" {
			return reason + " (embedded in " + ip.String() + ")"
		}
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 0:
			return "unspecified address" // 0.0.0.0/8 reaches the local host on some systems
		case ip4[0] >= 240:
			return "reserved address" // 240.0.0.0/4, including broadcast
		case ip4[0] == 198 && ip4[1]&0xfe == 18:
			return "reserved address" // 198.18.0.0/15, benchmarking
		}
	}
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
	if (ip.IsPrivate() || isCGNAT(ip)) && !privateTargetAllowed(ip) {
		return "private address"
	}
	return ""
}

var (
	nat64Prefix  = mustCIDR("64:ff9b::/96")
	nat64Local   = mustCIDR("64:ff9b:1::/48")
	sixToFour    = mustCIDR("2002::/16")
	teredoPrefix = mustCIDR("2001::/32")
)

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// embeddedIPv4 returns the IPv4 address an IPv6 transition address carries,
// or nil. IPv4-mapped addresses (::ffff:a.b.c.d) need no decoding: the net.IP
// methods already treat them as IPv4.
func embeddedIPv4(ip net.IP) net.IP {
	if ip.To4() != nil || len(ip) != net.IPv6len {
		return nil
	}
	switch {
	case nat64Prefix.Contains(ip), nat64Local.Contains(ip):
		return net.IPv4(ip[12], ip[13], ip[14], ip[15])
	case sixToFour.Contains(ip):
		return net.IPv4(ip[2], ip[3], ip[4], ip[5])
	case teredoPrefix.Contains(ip):
		// Teredo stores the client's IPv4 address inverted.
		return net.IPv4(^ip[12], ^ip[13], ^ip[14], ^ip[15])
	}
	return nil
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

// environmentProxy picks the proxy for a request; a variable so tests can set
// one (http.ProxyFromEnvironment reads the environment only once).
var environmentProxy = http.ProxyFromEnvironment

// safeProxy uses the proxy from the environment, if any. Through a proxy the
// dialer only sees the proxy's address, so the real target is checked here
// first. A name that resolves differently for the proxy than it did here can
// still get through, so the proxy should enforce its own egress rules too.
func safeProxy(req *http.Request) (*url.URL, error) {
	proxyURL, err := environmentProxy(req)
	if err != nil || proxyURL == nil {
		return proxyURL, err
	}
	if err := ValidateOutboundHost(req.URL.Hostname()); err != nil {
		return nil, err
	}
	return proxyURL, nil
}

// NewSecureHTTPClient returns an http.Client that enforces SSRF protections
func NewSecureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: safeCheckRedirect,
		Transport: &http.Transport{
			Proxy:                 safeProxy,
			DialContext:           SafeDialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
