package discovery

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNewmDNSScanner(t *testing.T) {
	scanner := NewmDNSScanner(5 * time.Second)
	if scanner == nil {
		t.Fatal("NewmDNSScanner returned nil")
	}
	if scanner.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", scanner.timeout)
	}
}

func TestMDNSScanner_Discover(t *testing.T) {
	scanner := NewmDNSScanner(1 * time.Second)

	// Test with empty network
	results, err := scanner.Discover(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty network, got nil")
	}

	// Test with invalid network
	results, err = scanner.Discover(context.Background(), "invalid")
	if err == nil {
		t.Error("Expected error for invalid network, got nil")
	}

	// Test with valid network (may not find anything)
	results, err = scanner.Discover(context.Background(), "127.0.0.0/8")
	// May not find anything in local network
	if err != nil {
		t.Logf("Discover returned error (may be expected): %v", err)
	}
	_ = results // Use results
}

func TestMDNSScanner_BuildmDNSQuery(t *testing.T) {
	scanner := NewmDNSScanner(5 * time.Second)

	query := scanner.buildmDNSQuery("_services._dns-sd._udp.local")
	if len(query) < 12 {
		t.Errorf("Expected query length >= 12, got %d", len(query))
	}

	// Check it's a query
	flags := (uint16(query[2]) << 8) | uint16(query[3])
	if flags&0x8000 != 0 {
		t.Error("Expected query (response bit should be 0)")
	}
}

func TestMDNSScanner_Parsename(t *testing.T) {
	scanner := NewmDNSScanner(5 * time.Second)

	// Test simple name
	data := []byte("\x03_test\x07_example\x03_com\x00")
	name, offset := scanner.parseName(data, 0)
	if name != "test.example.com" {
		t.Errorf("Expected test.example.com, got %s", name)
	}
	if offset != len(data) {
		t.Errorf("Expected offset %d, got %d", len(data), offset)
	}
}

func TestMDNSScanner_ExtractHostname(t *testing.T) {
	scanner := NewmDNSScanner(5 * time.Second)

	tests := []struct {
		name     string
		expected string
	}{
		{"host.local.", "host"},
		{"host.local", "host"},
		{"._http._tcp.local.", ""},
		{"host._http._tcp.local.", "host"},
		{"myhost", "myhost"},
	}

	for _, tt := range tests {
		result := scanner.extractHostname(tt.name)
		if result != tt.expected {
			t.Errorf("extractHostname(%s): expected %s, got %s", tt.name, tt.expected, result)
		}
	}
}

func TestMDNSScanner_GetServiceType(t *testing.T) {
	scanner := NewmDNSScanner(5 * time.Second)

	tests := []struct {
		name     string
		expected string
	}{
		{"_airplay._tcp", "Apple TV/AirPlay"},
		{"_afpovertcp._tcp", "File Sharing (AFP)"},
		{"_smb._tcp", "File Sharing (SMB)"},
		{"_ssh._tcp", "SSH"},
		{"_http._tcp", "Web Server"},
		{"_printer._tcp", "Printer"},
		{"_ipp._tcp", "Printer (IPP)"},
		{"_chromecast._tcp", "Chromecast"},
		{"_googlecast._tcp", "Google Cast"},
		{"_spotify-connect._tcp", "Spotify Connect"},
		{"_hap._tcp", "HomeKit"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		result := scanner.getServiceType(tt.name)
		if result != tt.expected {
			t.Errorf("getServiceType(%s): expected %s, got %s", tt.name, tt.expected, result)
		}
	}
}

func TestMDNSResult_Struct(t *testing.T) {
	result := mDNSResult{
		Hostname: "testhost",
		Type:     "SSH",
		IP:       "192.168.1.100",
	}

	if result.Hostname != "testhost" {
		t.Errorf("Expected hostname testhost, got %s", result.Hostname)
	}
	if result.Type != "SSH" {
		t.Errorf("Expected type SSH, got %s", result.Type)
	}
	if result.IP != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", result.IP)
	}
}

func TestMDNSScanner_ExtractIPFromAddr(t *testing.T) {
	scanner := NewmDNSScanner(5 * time.Second)

	// Test with UDPAddr
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5353}
	result := scanner.extractIPFromAddr(addr)
	if result != "192.168.1.100" {
		t.Errorf("Expected 192.168.1.100, got %s", result)
	}
}

func TestMDNSScanner_DiscoverWithContext(t *testing.T) {
	scanner := NewmDNSScanner(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test with cancelled context
	_, err := scanner.Discover(ctx, "192.168.1.0/24")
	if err != nil {
		t.Logf("Discover with cancelled context returned error (expected): %v", err)
	}
}
