package discovery

import (
	"testing"
)

func TestNewHostnameCorrelator(t *testing.T) {
	correlator := NewHostnameCorrelator()
	if correlator == nil {
		t.Fatal("NewHostnameCorrelator returned nil")
	}
}

func TestHostnameCorrelator_Correlate(t *testing.T) {
	correlator := NewHostnameCorrelator()

	// Test with empty sources
	conflict := correlator.Correlate([]HostnameSource{})
	if conflict != nil {
		t.Errorf("Expected nil for empty sources, got %+v", conflict)
	}

	// Test with single source
	sources := []HostnameSource{
		{Hostname: "testhost", Source: "dns", Confidence: 1},
	}
	conflict = correlator.Correlate(sources)
	if conflict == nil {
		t.Error("Expected conflict result for single source, got nil")
	}
	if conflict.Recommended != "testhost" {
		t.Errorf("Expected recommended testhost, got %s", conflict.Recommended)
	}

	// Test with multiple sources (same hostname)
	sources = []HostnameSource{
		{Hostname: "testhost.local", Source: "mdns", Confidence: 2},
		{Hostname: "testhost", Source: "dns", Confidence: 1},
	}
	conflict = correlator.Correlate(sources)
	if conflict == nil {
		t.Error("Expected conflict result, got nil")
	}
}

func TestHostnameCorrelator_NormalizeHostname(t *testing.T) {
	correlator := NewHostnameCorrelator()

	tests := []struct {
		input    string
		expected string
	}{
		{"TestHost", "testhost"},
		{"testhost.local", "testhost"},
		{"TESTHOST.LOCAL.", "testhost"},
		{"  TestHost  ", "testhost"},
		{"My-Test.Host", "my-test.host"},
	}

	for _, tt := range tests {
		result := correlator.normalizeHostname(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeHostname(%s): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func TestHostnameCorrelator_SelectBestHostname(t *testing.T) {
	correlator := NewHostnameCorrelator()

	uniqueHostnames := map[string][]HostnameSource{
		"testhost": {
			{Hostname: "testhost", Source: "dns", Confidence: 1},
			{Hostname: "testhost", Source: "ssh", Confidence: 3},
		},
	}

	// Should select highest confidence
	best := correlator.selectBestHostname(uniqueHostnames)
	if best != "testhost" {
		t.Errorf("Expected testhost, got %s", best)
	}
}

func TestHostnameCorrelator_HasConflicts(t *testing.T) {
	correlator := NewHostnameCorrelator()

	// Test with no conflicts
	sources := []HostnameSource{
		{Hostname: "testhost", Source: "dns", Confidence: 1},
	}
	conflict := correlator.Correlate(sources)
	if correlator.HasConflicts(conflict) {
		t.Error("Expected no conflicts for single source")
	}

	// Test with conflicts
	sources = []HostnameSource{
		{Hostname: "testhost.local", Source: "mdns", Confidence: 2},
		{Hostname: "testhost", Source: "dns", Confidence: 1},
	}
	conflict = correlator.Correlate(sources)
	if !correlator.HasConflicts(conflict) {
		t.Error("Expected conflicts for multiple sources")
	}
}

func TestHostnameCorrelator_GetPreferredSource(t *testing.T) {
	correlator := NewHostnameCorrelator()

	sources := []HostnameSource{
		{Hostname: "testhost", Source: "dns", Confidence: 1},
		{Hostname: "testhost", Source: "ssh", Confidence: 3},
		{Hostname: "testhost", Source: "snmp", Confidence: 3},
	}

	preferred := correlator.GetPreferredSource(sources)
	if preferred != "ssh" {
		t.Errorf("Expected ssh, got %s", preferred)
	}
}

func TestHostnameCorrelator_CompareSources(t *testing.T) {
	correlator := NewHostnameCorrelator()

	tests := []struct {
		name     string
		a        HostnameSource
		b        HostnameSource
		expected int
	}{
		{
			"higher confidence",
			HostnameSource{Hostname: "test", Source: "ssh", Confidence: 3},
			HostnameSource{Hostname: "test", Source: "dns", Confidence: 1},
			1,
		},
		{
			"same confidence, ssh priority",
			HostnameSource{Hostname: "test", Source: "ssh", Confidence: 3},
			HostnameSource{Hostname: "test", Source: "snmp", Confidence: 3},
			1,
		},
	}

	for _, tt := range tests {
		result := correlator.compareSources(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, result)
		}
	}
}

func TestHostnameCorrelator_MatchHostnames(t *testing.T) {
	correlator := NewHostnameCorrelator()

	tests := []struct {
		a        string
		b        string
		expected bool
	}{
		{"testhost", "testhost", true},
		{"testhost", "TestHost", true},
		{"testhost", "TESTHOST", true},
		{"testhost.local", "testhost", true},
		{"testhost.local.", "testhost", true},
		{"testhost", "otherhost", false},
		{"testhost", "testserver", false},
	}

	for _, tt := range tests {
		result := correlator.MatchHostnames(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("MatchHostnames(%s, %s): expected %v, got %v", tt.a, tt.b, tt.expected, result)
		}
	}
}

func TestHostnameConflict_Struct(t *testing.T) {
	conflict := HostnameConflict{
		Hostname:    "testhost",
		Sources:     []string{"dns", "ssh"},
		Conflicts:   map[string][]string{"testhost": {"dns", "ssh"}},
		Recommended: "testhost",
	}

	if conflict.Hostname != "testhost" {
		t.Errorf("Expected hostname testhost, got %s", conflict.Hostname)
	}
	if len(conflict.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(conflict.Sources))
	}
	if conflict.Recommended != "testhost" {
		t.Errorf("Recommended: expected testhost, got %s", conflict.Recommended)
	}
}
