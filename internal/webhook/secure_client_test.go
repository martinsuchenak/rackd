package webhook

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSafeDialContext_Allowed(t *testing.T) {
	// Start a local test server to act as a allowed external/internal service
	// Note: In typical environments httptest spins up on 127.0.0.1 which is blocked by our SSRF filter.
	// We'll need to mock the SafeDialContext logic or test it in a way that doesn't actually connect,
	// or we test the error explicitly.

	tests := []struct {
		name      string
		addr      string
		wantError string
	}{
		{
			name:      "Loopback IPv4",
			addr:      "127.0.0.1:80",
			wantError: "SSRF prevention",
		},
		{
			name:      "Localhost name",
			addr:      "localhost:80",
			wantError: "SSRF prevention",
		},
		{
			name:      "Loopback IPv6",
			addr:      "[::1]:80",
			wantError: "SSRF prevention",
		},
		{
			name:      "Unspecified IPv4",
			addr:      "0.0.0.0:80",
			wantError: "SSRF prevention",
		},
		{
			name:      "Cloud Metadata AWS",
			addr:      "169.254.169.254:80",
			wantError: "SSRF prevention",
		},
		{
			name:      "Valid External IP - no server listening so connection refused",
			addr:      "8.8.8.8:80",
			wantError: "connection refused", // Or timeout, but not SSRF blocked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := SafeDialContext(ctx, "tcp", tt.addr)
			if err == nil {
				t.Fatalf("expected error for address %s, but got nil", tt.addr)
			}

			// Some systems might resolve valid IPs to generic timeout errors.
			// We mainly care that our SSRF logic triggers the correct error.
			if tt.wantError == "SSRF prevention" {
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("expected error containing %q, got: %v", tt.wantError, err)
				}
			} else {
				if strings.Contains(err.Error(), "SSRF prevention") {
					t.Errorf("expected a normal network error, got SSRF block: %v", err)
				}
			}
		})
	}
}

func TestNewSecureHTTPClient(t *testing.T) {
	client := NewSecureHTTPClient(5 * time.Second)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", client.Timeout)
	}

	// Make an SSRF attempt with the full client
	_, err := client.Get("http://169.254.169.254/latest/meta-data")
	if err == nil {
		t.Fatal("expected request to cloud metadata to fail")
	}

	if !strings.Contains(err.Error(), "SSRF prevention") {
		t.Errorf("expected SSRF prevention error, got: %v", err)
	}
}

func TestBlockedIPReasonDecodesEmbeddedIPv4AndReservedRanges(t *testing.T) {
	SetPrivateRangePolicy(false)
	defer SetPrivateRangePolicy(false)
	for addr, blocked := range map[string]bool{
		"64:ff9b::a00:1":                 true,  // NAT64 for 10.0.0.1
		"64:ff9b::7f00:1":                true,  // NAT64 for 127.0.0.1
		"64:ff9b::a9fe:a9fe":             true,  // NAT64 for 169.254.169.254
		"2002:a00:1::1":                  true,  // 6to4 for 10.0.0.1
		"2001:0:4136:e378:0:0:80ff:fffe": true,  // Teredo for 127.0.0.1 (inverted)
		"64:ff9b::cb00:710a":             false, // NAT64 for 203.0.113.10, public
		"0.1.2.3":                        true,
		"198.18.0.1":                     true,
		"240.0.0.1":                      true,
		"255.255.255.255":                true,
		"203.0.113.10":                   false,
		"2001:db8::1":                    false,
	} {
		ip := net.ParseIP(addr)
		if ip == nil {
			t.Fatalf("bad test address %s", addr)
		}
		if got := blockedIPReason(ip) != ""; got != blocked {
			t.Errorf("%s: blocked = %v (%q), want %v", addr, got, blockedIPReason(ip), blocked)
		}
	}
}

// Through a proxy the dialer only sees the proxy, so the target is checked
// before the proxy is used.
func TestSafeProxyChecksTheTarget(t *testing.T) {
	proxyURL, _ := url.Parse("http://203.0.113.5:3128")
	environmentProxy = http.ProxyURL(proxyURL)
	defer func() { environmentProxy = http.ProxyFromEnvironment }()
	for target, ok := range map[string]bool{
		"https://169.254.169.254/latest/meta-data": false,
		"http://127.0.0.1:8080/":                   false,
		"https://203.0.113.10/hook":                true,
	} {
		req, _ := http.NewRequest(http.MethodPost, target, nil)
		proxy, err := safeProxy(req)
		if ok && (err != nil || proxy == nil) {
			t.Errorf("%s: proxy %v, err %v; want the proxy", target, proxy, err)
		}
		if !ok && err == nil {
			t.Errorf("%s: allowed through the proxy", target)
		}
	}
}

// SSRF_ALLOWED_TARGETS lifts the private-range block for the listed hosts
// and ranges only, and can't lift any other block.
func TestAllowedPrivateTargets(t *testing.T) {
	SetPrivateRangePolicy(false)
	defer SetAllowedPrivateTargets(nil)

	for _, entry := range []string{"0.0.0.0/0", "10.0.0.0/7", "127.0.0.1", "169.254.169.254", "8.8.8.8", "192.168.1.0/33", "nonsense"} {
		if _, err := ParseAllowedTargets([]string{entry}); err == nil {
			t.Errorf("%s: accepted, want an error", entry)
		}
	}
	nets, err := ParseAllowedTargets([]string{"192.168.88.51", " 10.20.0.0/24 ", "", "fd00::1"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	SetAllowedPrivateTargets(nets)

	for addr, blocked := range map[string]bool{
		"192.168.88.51":   false,
		"192.168.88.52":   true,
		"10.20.0.9":       false,
		"10.20.1.9":       true,
		"fd00::1":         false,
		"fd00::2":         true,
		"127.0.0.1":       true,
		"169.254.169.254": true,
		"203.0.113.10":    false,
	} {
		if got := blockedIPReason(net.ParseIP(addr)) != ""; got != blocked {
			t.Errorf("%s: blocked = %v, want %v", addr, got, blocked)
		}
	}

	SetAllowedPrivateTargets(nil)
	if blockedIPReason(net.ParseIP("192.168.88.51")) == "" {
		t.Error("192.168.88.51 allowed after clearing the list")
	}
}
