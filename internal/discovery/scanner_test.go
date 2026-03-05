package discovery

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	log.Init("text", "error", io.Discard)
}

func newTestUnifiedScanner(t *testing.T) (*UnifiedScanner, storage.ExtendedStorage) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	return NewUnifiedScanner(store, store, nil, 100*time.Millisecond, true), store
}

// Helper function tests

func TestCountHosts(t *testing.T) {
	tests := []struct {
		cidr     string
		expected int
	}{
		{"192.168.1.0/24", 256},
		{"192.168.1.0/30", 4},
		{"192.168.1.0/32", 1},
		{"10.0.0.0/16", 65536},
	}

	for _, tt := range tests {
		_, ipNet, err := net.ParseCIDR(tt.cidr)
		if err != nil {
			t.Fatalf("Failed to parse CIDR %s: %v", tt.cidr, err)
		}

		result := countHosts(ipNet)
		if result != tt.expected {
			t.Errorf("countHosts(%s) = %d, want %d", tt.cidr, result, tt.expected)
		}
	}
}

func TestExpandCIDR(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("192.168.1.0/30")
	ips := expandCIDR(ipNet)

	// /30 has 4 IPs, minus network and broadcast = 2 usable
	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(ips))
	}

	expected := []string{"192.168.1.1", "192.168.1.2"}
	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("IP[%d] = %s, want %s", i, ip, expected[i])
		}
	}
}

func TestExpandCIDR_SingleHost(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("192.168.1.1/32")
	ips := expandCIDR(ipNet)

	// /32 has 1 IP, no network/broadcast to remove
	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}
}

func TestIncrementIP(t *testing.T) {
	ip := net.ParseIP("192.168.1.1").To4()
	incrementIP(ip)

	if ip.String() != "192.168.1.2" {
		t.Errorf("Expected 192.168.1.2, got %s", ip.String())
	}
}

func TestIncrementIP_Overflow(t *testing.T) {
	ip := net.ParseIP("192.168.1.255").To4()
	incrementIP(ip)

	if ip.String() != "192.168.2.0" {
		t.Errorf("Expected 192.168.2.0, got %s", ip.String())
	}
}

func TestGetTop100Ports(t *testing.T) {
	ports := getTop100Ports()
	if len(ports) != 100 {
		t.Errorf("Expected 100 ports, got %d", len(ports))
	}

	// Check some common ports are included
	commonPorts := []int{22, 80, 443, 3306, 5432}
	for _, p := range commonPorts {
		found := false
		for _, port := range ports {
			if port == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected port %d to be in top 100", p)
		}
	}
}

// UnifiedScanner tests

func TestScan_InvalidCIDR(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test",
		Subnet: "invalid-cidr",
	}

	_, err := scanner.Scan(context.Background(), network, model.ScanTypeQuick)
	if err == nil {
		t.Error("Expected error for invalid CIDR")
	}
}

func TestScan_CreatesScanRecord(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test",
		Subnet: "192.168.1.0/30",
	}
	store.CreateNetwork(ctx, network)

	scan, err := scanner.Scan(context.Background(), network, model.ScanTypeQuick)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if scan.ID == "" {
		t.Error("Expected scan ID to be set")
	}
	if scan.NetworkID != network.ID {
		t.Errorf("Expected NetworkID %s, got %s", network.ID, scan.NetworkID)
	}
	if scan.ScanType != model.ScanTypeQuick {
		t.Errorf("Expected ScanType %s, got %s", model.ScanTypeQuick, scan.ScanType)
	}
}

func TestGetScanStatus_FromCache(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test",
		Subnet: "192.168.1.0/30",
	}
	store.CreateNetwork(ctx, network)

	scan, _ := scanner.Scan(context.Background(), network, model.ScanTypeQuick)

	status, err := scanner.GetScanStatus(context.Background(), scan.ID)
	if err != nil {
		t.Fatalf("GetScanStatus failed: %v", err)
	}
	if status.ID != scan.ID {
		t.Errorf("Expected scan ID %s, got %s", scan.ID, status.ID)
	}
}

func TestGetScanStatus_FromStorage(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	// Create network first (foreign key constraint)
	network := &model.Network{ID: "net-1", Name: "Test", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(ctx, network)

	scan := &model.DiscoveryScan{
		ID:        "scan-123",
		NetworkID: "net-1",
		Status:    model.ScanStatusCompleted,
		ScanType:  model.ScanTypeQuick,
	}
	store.CreateDiscoveryScan(ctx, scan)

	status, err := scanner.GetScanStatus(context.Background(), "scan-123")
	if err != nil {
		t.Fatalf("GetScanStatus failed: %v", err)
	}
	if status.ID != "scan-123" {
		t.Errorf("Expected scan ID scan-123, got %s", status.ID)
	}
}

func TestScan_SubnetTooLarge(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test",
		Subnet: "10.0.0.0/8", // Too large
	}

	_, err := scanner.Scan(context.Background(), network, model.ScanTypeQuick)
	if err != ErrSubnetTooLarge {
		t.Errorf("Expected ErrSubnetTooLarge, got %v", err)
	}
}

func TestScan_MaxAllowedSubnet(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test",
		Subnet: "10.0.0.0/16", // Max allowed
	}
	store.CreateNetwork(ctx, network)

	scan, err := scanner.Scan(context.Background(), network, model.ScanTypeQuick)
	if err != nil {
		t.Errorf("Expected /16 to be allowed, got error: %v", err)
	}
	if scan == nil {
		t.Error("Expected scan to be created")
	}
}

func TestCleanupCompletedScans(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()

	// Add a completed scan with old timestamp
	oldTime := time.Now().Add(-5 * time.Minute)
	scanner.mu.Lock()
	scanner.scans["old-scan"] = &model.DiscoveryScan{
		ID:          "old-scan",
		Status:      model.ScanStatusCompleted,
		CompletedAt: &oldTime,
	}
	// Add a recent completed scan
	recentTime := time.Now()
	scanner.scans["recent-scan"] = &model.DiscoveryScan{
		ID:          "recent-scan",
		Status:      model.ScanStatusCompleted,
		CompletedAt: &recentTime,
	}
	// Add a running scan
	scanner.scans["running-scan"] = &model.DiscoveryScan{
		ID:     "running-scan",
		Status: model.ScanStatusRunning,
	}
	scanner.mu.Unlock()

	scanner.cleanupCompletedScans()

	scanner.mu.RLock()
	defer scanner.mu.RUnlock()

	if _, ok := scanner.scans["old-scan"]; ok {
		t.Error("Expected old-scan to be cleaned up")
	}
	if _, ok := scanner.scans["recent-scan"]; !ok {
		t.Error("Expected recent-scan to be kept")
	}
	if _, ok := scanner.scans["running-scan"]; !ok {
		t.Error("Expected running-scan to be kept")
	}
}
