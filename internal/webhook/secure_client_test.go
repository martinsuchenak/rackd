package webhook

import (
	"context"
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
