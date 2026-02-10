package discovery

import (
	"testing"
)

func TestNewOUIDatabase(t *testing.T) {
	db := NewOUIDatabase()
	if db == nil {
		t.Fatal("NewOUIDatabase returned nil")
	}
	if db.entries == nil {
		t.Fatal("OUI entries map not initialized")
	}
}

func TestOUIDatabase_Lookup(t *testing.T) {
	db := NewOUIDatabase()

	// Test some known OUIs from the database
	tests := []struct {
		mac      string
		expected string
	}{
		{"00:0c:29:12:34:56", "VMware"},
		{"00:05:85:12:34:56", "Broadcom"},
		{"00:0b:cd:12:34:56", "3Com"},
		{"00:0d:b9:12:34:56", "ZyXEL"},
		{"00:0e:c6:12:34:56", "Cisco"},
		{"00:0f:ea:12:34:56", "Netgear"},
		{"00:10:18:12:34:56", "Broadcom"},
		{"00:10:db:12:34:56", "Dell"},
		{"00:11:43:12:34:56", "Cisco"},
		{"00:12:3f:12:34:56", "Intel Corporate"},
		{"00:13:20:12:34:56", "Cisco"},
		{"00:14:22:12:34:56", "Dell"},
		{"00:15:17:12:34:56", "Hewlett Packard"},
		{"00:16:35:12:34:56", "Hewlett Packard"},
		{"00:17:08:12:34:56", "Hewlett Packard"},
		{"00:21:28:12:34:56", "Dell"},
		{"00:22:fa:12:34:56", "Dell"},
		{"00:23:ae:12:34:56", "Dell"},
		{"00:26:9e:12:34:56", "Intel Corporate"},
		{"00:1b:21:12:34:56", "Intel Corporate"},
		{"00:a0:c9:12:34:56", "Intel Corporate"},
		{"3c:d9:2b:12:34:56", "Intel Corporate"},
		{"00:e0:4c:12:34:56", "Realtek"},
		{"00:1a:a0:12:34:56", "Realtek"},
		{"00:04:ac:12:34:56", "Dell"},
		{"00:00:00:12:34:56", "Unknown"},
		{"ff:ff:ff:12:34:56", "Broadcast"},
	}

	for _, tt := range tests {
		result := db.Lookup(tt.mac)
		if result != tt.expected {
			t.Errorf("Lookup(%s): expected %s, got %s", tt.mac, tt.expected, result)
		}
	}
}

func TestOUIDatabase_LookupUnknown(t *testing.T) {
	db := NewOUIDatabase()

	tests := []string{
		"AA:BB:CC:DD:EE:FF",
		"99:88:77:66:55:44",
	}

	for _, mac := range tests {
		result := db.Lookup(mac)
		if result != "" {
			t.Errorf("Lookup(%s): expected empty, got %s", mac, result)
		}
	}
}

func TestOUIDatabase_LookupInvalidMAC(t *testing.T) {
	db := NewOUIDatabase()

	tests := []string{
		"",
		"invalid",
		"00:11:22",
		"GG:HH:II:JJ:KK:LL",
	}

	for _, mac := range tests {
		result := db.Lookup(mac)
		// Invalid MAC should return empty or not crash
		if result == "" {
			t.Logf("Lookup(%s) returned empty (expected for invalid MAC)", mac)
		}
	}
}

func TestOUIDatabase_AddEntry(t *testing.T) {
	db := NewOUIDatabase()

	// Add a custom OUI
	db.AddEntry("aa:bb:cc", "TestVendor")

	result := db.Lookup("aa:bb:cc:dd:ee:ff")
	if result != "TestVendor" {
		t.Errorf("Expected TestVendor, got %s", result)
	}
}

func TestOUIDatabase_Count(t *testing.T) {
	db := NewOUIDatabase()

	count := db.Count()
	if count <= 0 {
		t.Errorf("Expected positive OUI count, got %d", count)
	}

	// Add entry and verify count increases
	db.AddEntry("aa:bb:cc", "TestVendor")
	newCount := db.Count()
	if newCount <= count {
		t.Errorf("Expected count to increase, got %d (was %d)", newCount, count)
	}
}

func TestOUIDatabase_LookupPartial(t *testing.T) {
	db := NewOUIDatabase()

	// Test that only first 3 octets are used
	tests := []struct {
		mac      string
		expected string
	}{
		{"00:0c:29:AA:BB:CC", "VMware"},
		{"00:0c:29:DD:EE:FF", "VMware"},
	}

	for _, tt := range tests {
		result := db.Lookup(tt.mac)
		if result != tt.expected {
			t.Errorf("Lookup(%s): expected %s, got %s", tt.mac, tt.expected, result)
		}
	}
}
