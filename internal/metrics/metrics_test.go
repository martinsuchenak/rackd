package metrics

import (
	"testing"
	"time"
)

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	m.RecordHTTPRequest("GET", "/api/devices", 200, 50*time.Millisecond)
	m.RecordHTTPRequest("GET", "/api/devices", 200, 100*time.Millisecond)
	m.RecordHTTPRequest("POST", "/api/devices", 201, 75*time.Millisecond)

	if m.httpRequestsTotal.Load() != 3 {
		t.Errorf("Expected 3 requests, got %d", m.httpRequestsTotal.Load())
	}

	// Check histogram
	h := m.getOrCreateHistogram(&m.httpRequestDuration, "GET /api/devices")
	if h.count.Load() != 2 {
		t.Errorf("Expected 2 GET requests, got %d", h.count.Load())
	}
	avg := h.avg()
	if avg < 70 || avg > 80 {
		t.Errorf("Expected avg ~75ms, got %.2f", avg)
	}
}

func TestMetrics_RecordDiscoveryScan(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	m.RecordDiscoveryScan("ping", 5*time.Second, 10)
	m.RecordDiscoveryScan("ping", 3*time.Second, 5)

	if m.discoveryScans.Load() != 2 {
		t.Errorf("Expected 2 scans, got %d", m.discoveryScans.Load())
	}

	h := m.getOrCreateHistogram(&m.discoveryDuration, "ping")
	if h.count.Load() != 2 {
		t.Errorf("Expected 2 ping scans, got %d", h.count.Load())
	}
}

func TestMetrics_SetCounts(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	m.SetDeviceCount(100)
	m.SetNetworkCount(50)
	m.SetDatacenterCount(5)

	if m.devicesTotal.Load() != 100 {
		t.Errorf("Expected 100 devices, got %d", m.devicesTotal.Load())
	}
	if m.networksTotal.Load() != 50 {
		t.Errorf("Expected 50 networks, got %d", m.networksTotal.Load())
	}
	if m.datacentersTotal.Load() != 5 {
		t.Errorf("Expected 5 datacenters, got %d", m.datacentersTotal.Load())
	}
}

func TestMetrics_Export(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	m.RecordHTTPRequest("GET", "/api/devices", 200, 50*time.Millisecond)
	m.SetDeviceCount(100)

	output := m.Export()

	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Check for key metrics
	tests := []string{
		"http_requests_total",
		"devices_total",
		"go_goroutines",
		"process_uptime_seconds",
	}

	for _, test := range tests {
		if !contains(output, test) {
			t.Errorf("Expected output to contain %s", test)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
