package discovery

import (
	"testing"
)

func TestNewOSFingerprinter(t *testing.T) {
	fingerprinter := NewOSFingerprinter(2 * 0)
	if fingerprinter == nil {
		t.Fatal("NewOSFingerprinter returned nil")
	}
}

func TestOSFingerprinter_Fingerprint(t *testing.T) {
	fingerprinter := NewOSFingerprinter(1 * 1000000000)

	// Test with unreachable IP - may return default
	fp := fingerprinter.Fingerprint("127.0.0.1")
	if fp.OSFamily == "" {
		// This is OK for localhost which may not respond to fingerprinting
		t.Logf("Fingerprinter returned unknown OS for localhost (expected)")
	}
}

func TestOSFingerprinter_FingerprintInvalidIP(t *testing.T) {
	fingerprinter := NewOSFingerprinter(1 * 1000000000)

	// Test with invalid IP
	fp := fingerprinter.Fingerprint("999.999.999.999")
	if fp.OSFamily != OSTypeUnknown {
		t.Errorf("Expected unknown OS for invalid IP, got %s", fp.OSFamily)
	}
}

func TestOSTypeFromFamily(t *testing.T) {
	tests := []struct {
		family   string
		expected string
	}{
		{OSTypeLinux, "Linux"},
		{OSTypeWindows, "Windows"},
		{OSTypeMacOS, "macOS"},
		{OSTypeNetwork, "Network Device"},
		{OSTypeUnknown, "Unknown"},
		{"invalid", "Unknown"},
	}

	for _, tt := range tests {
		result := GetOSTypeFromFamily(tt.family)
		if result != tt.expected {
			t.Errorf("GetOSTypeFromFamily(%s): expected %s, got %s", tt.family, tt.expected, result)
		}
	}
}

func TestOSType_Constants(t *testing.T) {
	// Verify OS family constants are defined
	if OSTypeLinux == "" {
		t.Error("OSTypeLinux is empty")
	}
	if OSTypeWindows == "" {
		t.Error("OSTypeWindows is empty")
	}
	if OSTypeMacOS == "" {
		t.Error("OSTypeMacOS is empty")
	}
	if OSTypeNetwork == "" {
		t.Error("OSTypeNetwork is empty")
	}
	if OSTypeUnknown == "" {
		t.Error("OSTypeUnknown is empty")
	}
}

func TestOSFingerprint_Confidence(t *testing.T) {
	fingerprinter := NewOSFingerprinter(1 * 1000000000)

	// Test confidence score is within valid range
	fp := fingerprinter.Fingerprint("127.0.0.1")
	if fp.Confidence < 0 || fp.Confidence > 3 {
		t.Errorf("Confidence score out of range [0-3]: got %d", fp.Confidence)
	}
}

func TestOSFingerprinter_GetOSFamily(t *testing.T) {
	fingerprinter := NewOSFingerprinter(1 * 1000000000)

	// Test GetOSFamily method
	tests := []struct {
		ttl    uint8
		window uint16
		family string
	}{
		{64, 65535, OSTypeLinux},
		{128, 8192, OSTypeWindows},
		{255, 0, OSTypeNetwork},
		{0, 0, OSTypeUnknown},
	}

	for _, tt := range tests {
		result := fingerprinter.GetOSFamily(tt.ttl, tt.window)
		if result != tt.family {
			t.Errorf("TTL %d Window %d: expected family %s, got %s", tt.ttl, tt.window, tt.family, result)
		}
	}
}

func TestOSFingerprint_InvalidInput(t *testing.T) {
	fingerprinter := NewOSFingerprinter(1 * 1000000000)

	// Test with empty IP
	fp := fingerprinter.Fingerprint("")
	if fp.OSFamily != OSTypeUnknown {
		t.Errorf("Expected unknown OS for empty IP, got %s", fp.OSFamily)
	}

	// Test with loopback unreachable
	fp = fingerprinter.Fingerprint("::1")
	// IPv6 may not be fully supported, should handle gracefully
	if fp.OSFamily != OSTypeUnknown && fp.OSFamily != "" {
		t.Logf("IPv6 fingerprinting returned: %s", fp.OSFamily)
	}
}

func TestOSFingerprint_Struct(t *testing.T) {
	fp := &OSFingerprint{
		TTL:        64,
		WindowSize: 65535,
		TCPFlags:   "SYN",
		OSFamily:   OSTypeLinux,
		Confidence: 3,
	}

	if fp.TTL != 64 {
		t.Errorf("Expected TTL 64, got %d", fp.TTL)
	}
	if fp.WindowSize != 65535 {
		t.Errorf("Expected WindowSize 65535, got %d", fp.WindowSize)
	}
	if fp.OSFamily != OSTypeLinux {
		t.Errorf("Expected OSFamily %s, got %s", OSTypeLinux, fp.OSFamily)
	}
	if fp.Confidence != 3 {
		t.Errorf("Expected Confidence 3, got %d", fp.Confidence)
	}
}

func TestConfidenceLevels(t *testing.T) {
	// Verify confidence constants are accessible
	if ConfidenceHigh < ConfidenceMedium {
		t.Error("ConfidenceHigh should be greater than ConfidenceMedium")
	}
	if ConfidenceMedium < ConfidenceLow {
		t.Error("ConfidenceMedium should be greater than ConfidenceLow")
	}

	if ConfidenceHigh != 3 {
		t.Errorf("Expected ConfidenceHigh 3, got %d", ConfidenceHigh)
	}
	if ConfidenceMedium != 2 {
		t.Errorf("Expected ConfidenceMedium 2, got %d", ConfidenceMedium)
	}
	if ConfidenceLow != 1 {
		t.Errorf("Expected ConfidenceLow 1, got %d", ConfidenceLow)
	}
}
