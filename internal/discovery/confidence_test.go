package discovery

import (
	"testing"
)

func TestNewConfidenceScorer(t *testing.T) {
	scorer := NewConfidenceScorer()
	if scorer == nil {
		t.Fatal("NewConfidenceScorer returned nil")
	}
	if scorer.sources == nil {
		t.Fatal("Sources slice not initialized")
	}
}

func TestConfidenceScorer_Add(t *testing.T) {
	scorer := NewConfidenceScorer()

	scorer.Add("testhost", "dns", ConfidenceLow)
	if len(scorer.sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(scorer.sources))
	}

	scorer.Add("", "ssh", ConfidenceHigh)
	// Empty hostname should not be added
	if len(scorer.sources) != 1 {
		t.Errorf("Expected 1 source (empty hostname should not be added), got %d", len(scorer.sources))
	}
}

func TestConfidenceScorer_GetBest(t *testing.T) {
	scorer := NewConfidenceScorer()

	// Test with empty scorer
	hostname, confidence := scorer.GetBest()
	if hostname != "" {
		t.Errorf("Expected empty hostname, got %s", hostname)
	}
	if confidence != 0 {
		t.Errorf("Expected 0 confidence, got %d", confidence)
	}

	// Add sources
	scorer.Add("test1", "dns", ConfidenceLow)
	scorer.Add("test2", "ssh", ConfidenceHigh)

	hostname, confidence = scorer.GetBest()
	if hostname != "test2" {
		t.Errorf("Expected test2, got %s", hostname)
	}
	if confidence != ConfidenceHigh {
		t.Errorf("Expected confidence %d, got %d", ConfidenceHigh, confidence)
	}
}

func TestConfidenceScorer_GetAll(t *testing.T) {
	scorer := NewConfidenceScorer()

	scorer.Add("test1", "dns", ConfidenceLow)
	scorer.Add("test2", "ssh", ConfidenceHigh)
	scorer.Add("test3", "snmp", ConfidenceHigh)

	sources := scorer.GetAll()
	if len(sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(sources))
	}
}

func TestGetHostnameSourceConfidence(t *testing.T) {
	tests := []struct {
		source   string
		expected int
	}{
		{"ssh", ConfidenceHigh},
		{"snmp", ConfidenceHigh},
		{"netbios", ConfidenceMedium},
		{"mdns", ConfidenceMedium},
		{"dns", ConfidenceLow},
		{"unknown", ConfidenceLow},
	}

	for _, tt := range tests {
		result := GetHostnameSourceConfidence(tt.source)
		if result != tt.expected {
			t.Errorf("GetHostnameSourceConfidence(%s): expected %d, got %d", tt.source, tt.expected, result)
		}
	}
}

func TestHostnameSource_Struct(t *testing.T) {
	source := HostnameSource{
		Hostname:   "testhost",
		Source:     "ssh",
		Confidence: 3,
	}

	if source.Hostname != "testhost" {
		t.Errorf("Expected hostname testhost, got %s", source.Hostname)
	}
	if source.Source != "ssh" {
		t.Errorf("Expected source ssh, got %s", source.Source)
	}
	if source.Confidence != 3 {
		t.Errorf("Expected confidence 3, got %d", source.Confidence)
	}
}

func TestNewOSConfidenceScorer(t *testing.T) {
	scorer := NewOSConfidenceScorer()
	if scorer == nil {
		t.Fatal("NewOSConfidenceScorer returned nil")
	}
	if scorer.sources == nil {
		t.Fatal("OS sources slice not initialized")
	}
}

func TestOSConfidenceScorer_Add(t *testing.T) {
	scorer := NewOSConfidenceScorer()

	scorer.Add("Linux", "fingerprinting", ConfidenceHigh)
	if len(scorer.sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(scorer.sources))
	}

	scorer.Add("", "ssh", ConfidenceMedium)
	if len(scorer.sources) != 1 {
		t.Errorf("Expected 1 source (empty OS should not be added), got %d", len(scorer.sources))
	}
}

func TestOSConfidenceScorer_GetBest(t *testing.T) {
	scorer := NewOSConfidenceScorer()

	// Test with empty scorer
	os, confidence := scorer.GetBest()
	if os != "" {
		t.Errorf("Expected empty OS, got %s", os)
	}
	if confidence != 0 {
		t.Errorf("Expected 0 confidence, got %d", confidence)
	}

	// Add sources
	scorer.Add("Linux", "fingerprinting", ConfidenceHigh)
	scorer.Add("Windows", "snmp", ConfidenceLow)

	os, confidence = scorer.GetBest()
	if os != "Linux" {
		t.Errorf("Expected Linux, got %s", os)
	}
	if confidence != ConfidenceHigh {
		t.Errorf("Expected confidence %d, got %d", ConfidenceHigh, confidence)
	}
}

func TestGetOSSourceConfidence(t *testing.T) {
	tests := []struct {
		source   string
		expected int
	}{
		{"fingerprinting", ConfidenceHigh},
		{"ssh", ConfidenceMedium},
		{"snmp", ConfidenceLow},
		{"unknown", ConfidenceLow},
	}

	for _, tt := range tests {
		result := GetOSSourceConfidence(tt.source)
		if result != tt.expected {
			t.Errorf("GetOSSourceConfidence(%s): expected %d, got %d", tt.source, tt.expected, result)
		}
	}
}

func TestOSSource_Struct(t *testing.T) {
	source := OSSource{
		OS:         "Linux",
		Source:     "fingerprinting",
		Confidence: 3,
	}

	if source.OS != "Linux" {
		t.Errorf("Expected OS Linux, got %s", source.OS)
	}
	if source.Source != "fingerprinting" {
		t.Errorf("Expected source fingerprinting, got %s", source.Source)
	}
	if source.Confidence != 3 {
		t.Errorf("Expected confidence 3, got %d", source.Confidence)
	}
}
