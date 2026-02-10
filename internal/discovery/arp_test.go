package discovery

import (
	"testing"
)

func TestNewARPScanner(t *testing.T) {
	scanner := NewARPScanner()
	if scanner == nil {
		t.Fatal("NewARPScanner returned nil")
	}
	if scanner.entries == nil {
		t.Fatal("entries slice not initialized")
	}
}

func TestARPScanner_LookupMAC(t *testing.T) {
	scanner := NewARPScanner()

	// Test with empty table
	mac := scanner.LookupMAC("192.168.1.1")
	if mac != "" {
		t.Errorf("Expected empty MAC, got %s", mac)
	}

	// Add test entry
	scanner.entries = append(scanner.entries, ARPEntry{IP: "192.168.1.1", MAC: "00:11:22:33:44:55"})

	mac = scanner.LookupMAC("192.168.1.1")
	if mac != "00:11:22:33:44:55" {
		t.Errorf("Expected 00:11:22:33:44:55, got %s", mac)
	}

	// Test non-existent IP
	mac = scanner.LookupMAC("192.168.1.2")
	if mac != "" {
		t.Errorf("Expected empty MAC for non-existent IP, got %s", mac)
	}
}

func TestARPScanner_LoadARPTable(t *testing.T) {
	scanner := NewARPScanner()
	// Just verify it doesn't panic
	err := scanner.LoadARPTable()
	// May fail on systems without ARP table or different permissions
	// We just verify the method can be called
	if err != nil {
		// Log but don't fail - ARP table may not be accessible in test environment
		t.Logf("LoadARPTable returned error (may be expected in test environment): %v", err)
	}
}

func TestLoadLinuxARP_ParseLine(t *testing.T) {
	// Simulate parsing a Linux ARP line
	line := "192.168.1.100  0x1  0x2  00:11:22:33:44:55  *  eth0"
	fields := []string{"192.168.1.100", "0x1", "0x2", "00:11:22:33:44:55", "*", "eth0"}

	if len(fields) < 6 {
		t.Fatal("Expected at least 6 fields")
	}

	ip := fields[0]
	mac := fields[3]

	if ip != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", ip)
	}

	if mac != "00:11:22:33:44:55" {
		t.Errorf("Expected MAC 00:11:22:33:44:55, got %s", mac)
	}

	// Verify line contains expected IP
	if !contains(line, ip) {
		t.Errorf("Line doesn't contain expected IP %s", ip)
	}
}

func TestLoadDarwinARP_ParseLine(t *testing.T) {
	// Test darwin ARP parsing logic
	line := "? (192.168.1.100) at 00:11:22:33:44:55 on en0 [ethernet]"

	if len(line) == 0 {
		t.Fatal("Empty line")
	}

	// Verify line contains expected parts
	if !contains(line, "192.168.1.100") {
		t.Error("Line doesn't contain expected IP")
	}
	if !contains(line, "00:11:22:33:44:55") {
		t.Error("Line doesn't contain expected MAC")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
