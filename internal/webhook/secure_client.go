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
		// Block unspecified (0.0.0.0)
		// Block link-local metadata (169.254.x.x)
		if ip.IsLoopback() || ip.IsUnspecified() || (ip.To4() != nil && ip.To4()[0] == 169 && ip.To4()[1] == 254) {
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

// NewSecureHTTPClient returns an http.Client that enforces SSRF protections
func NewSecureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
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
