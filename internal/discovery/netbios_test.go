package discovery

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNewNetBIOSScanner(t *testing.T) {
	scanner := NewNetBIOSScanner(5 * time.Second)
	if scanner == nil {
		t.Fatal("NewNetBIOSScanner returned nil")
	}
	if scanner.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", scanner.timeout)
	}
}

func TestNetBIOSScanner_Discover(t *testing.T) {
	scanner := NewNetBIOSScanner(1 * time.Second)

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

func TestNetBIOSScanner_BuildNBNSQuery(t *testing.T) {
	scanner := NewNetBIOSScanner(5 * time.Second)

	query := scanner.buildNBNSQuery()
	if len(query) < 12 {
		t.Errorf("Expected query length >= 12, got %d", len(query))
	}

	// Check transaction ID is 0
	if query[0] != 0 || query[1] != 0 {
		t.Error("Expected transaction ID 0")
	}
}

func TestEncodeNetBIOSName(t *testing.T) {
	tests := []struct {
		name     string
		expected int // expected length
	}{
		{"*", 32},
		{"TEST", 32},
		{"LONGNAME123456", 32},
	}

	for _, tt := range tests {
		result := encodeNetBIOSName(tt.name)
		if len(result) != tt.expected {
			t.Errorf("encodeNetBIOSName(%s): expected length %d, got %d", tt.name, tt.expected, len(result))
		}
	}
}

func TestDecodeNetBIOSName(t *testing.T) {
	tests := []struct {
		encoded  []byte
		expected string
	}{
		{[]byte("CKFDENECHECAHEHECEJ"), "TEST"},
	}

	for _, tt := range tests {
		result := decodeNetBIOSName(tt.encoded)
		if result != tt.expected {
			t.Errorf("decodeNetBIOSName(%s): expected %s, got %s", tt.encoded, tt.expected, result)
		}
	}
}

func TestNetBIOSScanner_ParseNBNSResponse(t *testing.T) {
	scanner := NewNetBIOSScanner(5 * time.Second)

	// Test with empty data
	result := scanner.parseNBNSResponse([]byte{})
	if result != "" {
		t.Errorf("Expected empty hostname for empty data, got %s", result)
	}

	// Test with too short data
	result = scanner.parseNBNSResponse([]byte{1, 2, 3})
	if result != "" {
		t.Errorf("Expected empty hostname for short data, got %s", result)
	}
}

func TestNetBIOSScanner_GetBroadcastAddr(t *testing.T) {
	scanner := NewNetBIOSScanner(5 * time.Second)

	ip := net.ParseIP("192.168.1.10")
	_, ipNet, _ := net.ParseCIDR("192.168.1.0/24")

	broadcast := scanner.getBroadcastAddr(ip, ipNet)
	if broadcast == nil {
		t.Fatal("Expected broadcast address, got nil")
	}

	expected := net.ParseIP("192.168.1.255")
	if !broadcast.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected, broadcast)
	}
}

func TestNetBIOSResult_Struct(t *testing.T) {
	result := NetBIOSResult{
		Hostname: "TESTCOMPUTER",
		IP:       "192.168.1.100",
	}

	if result.Hostname != "TESTCOMPUTER" {
		t.Errorf("Expected hostname TESTCOMPUTER, got %s", result.Hostname)
	}
	if result.IP != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", result.IP)
	}
}

func TestNetBIOSScanner_DiscoverWithContext(t *testing.T) {
	scanner := NewNetBIOSScanner(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test with cancelled context
	_, err := scanner.Discover(ctx, "192.168.1.0/24")
	if err != nil {
		t.Logf("Discover with cancelled context returned error (expected): %v", err)
	}
}
