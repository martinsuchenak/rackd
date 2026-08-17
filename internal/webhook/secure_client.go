package webhook

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

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
		// Block loopback (e.g., 127.0.0.1, ::1)
		// Block unspecified (0.0.0.0, ::)
		// Block link-local unicast/multicast — covers the cloud metadata range
		// 169.254.0.0/16 AND the IPv6 fe80::/10 range (previously missed).
		// Block multicast globally.
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
			return nil, fmt.Errorf("SSRF prevention: connection to %s blocked (restricted IP %s)", host, ip.String())
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
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
			return fmt.Errorf("SSRF prevention: redirect to %s blocked (restricted IP %s)", host, ip.String())
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
