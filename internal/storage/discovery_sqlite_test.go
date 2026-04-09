package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Discovery Storage Tests
// ============================================================================

func TestDiscoveredDeviceCRUD(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Create network first
	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create discovered device
	device := &model.DiscoveredDevice{
		IP:         "192.168.1.100",
		MACAddress: "00:11:22:33:44:55",
		Hostname:   "test-host",
		NetworkID:  network.ID,
		Status:     "active",
		Confidence: 80,
		OSGuess:    "Linux",
		Vendor:     "Dell",
		OpenPorts:  []int{22, 80, 443},
		Services: []model.ServiceInfo{
			{Port: 22, Protocol: "tcp", Service: "ssh", Version: "OpenSSH 8.0"},
		},
	}
	if err := storage.CreateDiscoveredDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDiscoveredDevice failed: %v", err)
	}
	if device.ID == "" {
		t.Error("device ID should be set")
	}

	// Get device
	got, err := storage.GetDiscoveredDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetDiscoveredDevice failed: %v", err)
	}
	if got.IP != device.IP || got.Hostname != device.Hostname {
		t.Errorf("device mismatch: got %+v", got)
	}
	if len(got.OpenPorts) != 3 || got.OpenPorts[0] != 22 {
		t.Errorf("open_ports mismatch: got %v", got.OpenPorts)
	}
	if len(got.Services) != 1 || got.Services[0].Service != "ssh" {
		t.Errorf("services mismatch: got %v", got.Services)
	}

	// Update device
	device.Hostname = "updated-host"
	device.Confidence = 95
	if err := storage.UpdateDiscoveredDevice(context.Background(), device); err != nil {
		t.Fatalf("UpdateDiscoveredDevice failed: %v", err)
	}
	got, _ = storage.GetDiscoveredDevice(context.Background(), device.ID)
	if got.Hostname != "updated-host" || got.Confidence != 95 {
		t.Errorf("update failed: got %+v", got)
	}

	// Delete device
	if err := storage.DeleteDiscoveredDevice(context.Background(), device.ID); err != nil {
		t.Fatalf("DeleteDiscoveredDevice failed: %v", err)
	}
	_, err = storage.GetDiscoveredDevice(context.Background(), device.ID)
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}
}

func TestDiscoveredDeviceByIP(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	device := &model.DiscoveredDevice{
		IP:        "192.168.1.50",
		NetworkID: network.ID,
		Status:    "active",
	}
	storage.CreateDiscoveredDevice(context.Background(), device)

	got, err := storage.GetDiscoveredDeviceByIP(context.Background(), network.ID, "192.168.1.50")
	if err != nil {
		t.Fatalf("GetDiscoveredDeviceByIP failed: %v", err)
	}
	if got.ID != device.ID {
		t.Errorf("device ID mismatch")
	}

	// Not found
	_, err = storage.GetDiscoveredDeviceByIP(context.Background(), network.ID, "192.168.1.99")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}
}

func TestListDiscoveredDevices(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network1 := &model.Network{Name: "Net1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Net2", Subnet: "192.168.2.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{IP: "192.168.1.1", NetworkID: network1.ID})
	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{IP: "192.168.1.2", NetworkID: network1.ID})
	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{IP: "192.168.2.1", NetworkID: network2.ID})

	// List all
	all, err := storage.ListDiscoveredDevices(context.Background(), "")
	if err != nil {
		t.Fatalf("ListDiscoveredDevices failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 devices, got %d", len(all))
	}

	// List by network
	net1Devices, err := storage.ListDiscoveredDevices(context.Background(), network1.ID)
	if err != nil {
		t.Fatalf("ListDiscoveredDevices failed: %v", err)
	}
	if len(net1Devices) != 2 {
		t.Errorf("expected 2 devices for network1, got %d", len(net1Devices))
	}
}

func TestPromoteDiscoveredDevice(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	discovered := &model.DiscoveredDevice{IP: "192.168.1.10", NetworkID: network.ID}
	storage.CreateDiscoveredDevice(context.Background(), discovered)

	device := &model.Device{Name: "Promoted Device"}
	storage.CreateDevice(context.Background(), device)

	if err := storage.PromoteDiscoveredDevice(context.Background(), discovered.ID, device.ID); err != nil {
		t.Fatalf("PromoteDiscoveredDevice failed: %v", err)
	}

	got, _ := storage.GetDiscoveredDevice(context.Background(), discovered.ID)
	if got.PromotedToDeviceID != device.ID {
		t.Errorf("promoted_to_device_id mismatch: got %s", got.PromotedToDeviceID)
	}
	if got.PromotedAt == nil {
		t.Error("promoted_at should be set")
	}
}

func TestDiscoveryScanCRUD(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	scan := &model.DiscoveryScan{
		NetworkID:  network.ID,
		Status:     model.ScanStatusPending,
		ScanType:   model.ScanTypeFull,
		TotalHosts: 254,
	}
	if err := storage.CreateDiscoveryScan(context.Background(), scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}
	if scan.ID == "" {
		t.Error("scan ID should be set")
	}

	got, err := storage.GetDiscoveryScan(context.Background(), scan.ID)
	if err != nil {
		t.Fatalf("GetDiscoveryScan failed: %v", err)
	}
	if got.Status != model.ScanStatusPending || got.TotalHosts != 254 {
		t.Errorf("scan mismatch: got %+v", got)
	}

	// Update scan
	scan.Status = model.ScanStatusRunning
	scan.ScannedHosts = 50
	scan.ProgressPercent = 19.7
	if err := storage.UpdateDiscoveryScan(context.Background(), scan); err != nil {
		t.Fatalf("UpdateDiscoveryScan failed: %v", err)
	}
	got, _ = storage.GetDiscoveryScan(context.Background(), scan.ID)
	if got.Status != model.ScanStatusRunning || got.ScannedHosts != 50 {
		t.Errorf("update failed: got %+v", got)
	}
}

func TestListDiscoveryScans(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	storage.CreateDiscoveryScan(context.Background(), &model.DiscoveryScan{NetworkID: network.ID, Status: model.ScanStatusCompleted})
	storage.CreateDiscoveryScan(context.Background(), &model.DiscoveryScan{NetworkID: network.ID, Status: model.ScanStatusRunning})

	scans, err := storage.ListDiscoveryScans(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("ListDiscoveryScans failed: %v", err)
	}
	if len(scans) != 2 {
		t.Errorf("expected 2 scans, got %d", len(scans))
	}
}

func TestDiscoveryRuleCRUD(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	rule := &model.DiscoveryRule{
		NetworkID:     network.ID,
		Enabled:       true,
		ScanType:      model.ScanTypeFull,
		IntervalHours: 24,
		ExcludeIPs:    "192.168.1.1,192.168.1.254",
	}
	if err := storage.SaveDiscoveryRule(context.Background(), rule); err != nil {
		t.Fatalf("SaveDiscoveryRule failed: %v", err)
	}

	got, err := storage.GetDiscoveryRuleByNetwork(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetDiscoveryRuleByNetwork failed: %v", err)
	}
	if !got.Enabled || got.IntervalHours != 24 {
		t.Errorf("rule mismatch: got %+v", got)
	}

	// Update rule (upsert)
	rule.Enabled = false
	rule.IntervalHours = 12
	if err := storage.SaveDiscoveryRule(context.Background(), rule); err != nil {
		t.Fatalf("SaveDiscoveryRule update failed: %v", err)
	}
	got, _ = storage.GetDiscoveryRuleByNetwork(context.Background(), network.ID)
	if got.Enabled || got.IntervalHours != 12 {
		t.Errorf("update failed: got %+v", got)
	}
}

func TestListDiscoveryRules(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network1 := &model.Network{Name: "Net1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Net2", Subnet: "192.168.2.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	storage.SaveDiscoveryRule(context.Background(), &model.DiscoveryRule{NetworkID: network1.ID, Enabled: true})
	storage.SaveDiscoveryRule(context.Background(), &model.DiscoveryRule{NetworkID: network2.ID, Enabled: false})

	rules, err := storage.ListDiscoveryRules(context.Background())
	if err != nil {
		t.Fatalf("ListDiscoveryRules failed: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
}

func TestCleanupOldDiscoveries(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create devices - one will be promoted
	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{IP: "192.168.1.1", NetworkID: network.ID})
	promoted := &model.DiscoveredDevice{IP: "192.168.1.2", NetworkID: network.ID}
	storage.CreateDiscoveredDevice(context.Background(), promoted)

	device := &model.Device{Name: "Promoted"}
	storage.CreateDevice(context.Background(), device)
	storage.PromoteDiscoveredDevice(context.Background(), promoted.ID, device.ID)

	// Cleanup with 0 days should remove non-promoted devices
	if err := storage.CleanupOldDiscoveries(context.Background(), 0); err != nil {
		t.Fatalf("CleanupOldDiscoveries failed: %v", err)
	}

	devices, _ := storage.ListDiscoveredDevices(context.Background(), network.ID)
	if len(devices) != 1 {
		t.Errorf("expected 1 device (promoted), got %d", len(devices))
	}
	if devices[0].PromotedToDeviceID == "" {
		t.Error("remaining device should be the promoted one")
	}
}

func TestDiscoveryNotFoundErrors(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	_, err = storage.GetDiscoveredDevice(context.Background(), "nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	_, err = storage.GetDiscoveryScan(context.Background(), "nonexistent")
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}

	_, err = storage.GetDiscoveryRule(context.Background(), "nonexistent")
	if err != ErrRuleNotFound {
		t.Errorf("expected ErrRuleNotFound, got %v", err)
	}

	err = storage.UpdateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{ID: "nonexistent"})
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	err = storage.DeleteDiscoveredDevice(context.Background(), "nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	err = storage.PromoteDiscoveredDevice(context.Background(), "nonexistent", "device-id")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	err = storage.UpdateDiscoveryScan(context.Background(), &model.DiscoveryScan{ID: "nonexistent"})
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}
}

func TestDiscoveryInvalidIDs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// UpdateDiscoveredDevice with non-existent ID
	err := storage.UpdateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{ID: "nonexistent", IP: "192.168.1.1", NetworkID: network.ID})
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// GetDiscoveredDevice with non-existent ID returns not found
	_, err = storage.GetDiscoveredDevice(context.Background(), "nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// DeleteDiscoveredDevice with non-existent ID returns not found
	err = storage.DeleteDiscoveredDevice(context.Background(), "nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// PromoteDiscoveredDevice with non-existent discovered ID
	err = storage.PromoteDiscoveredDevice(context.Background(), "nonexistent", "device")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// GetDiscoveredDeviceByIP with non-existent network returns not found
	_, err = storage.GetDiscoveredDeviceByIP(context.Background(), "nonexistent", "192.168.1.1")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}
}

func TestDiscoveryScanInvalidIDs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// UpdateDiscoveryScan with non-existent ID
	err := storage.UpdateDiscoveryScan(context.Background(), &model.DiscoveryScan{ID: "nonexistent", NetworkID: network.ID})
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}

	// GetDiscoveryScan with non-existent ID
	_, err = storage.GetDiscoveryScan(context.Background(), "nonexistent")
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}
}

func TestDiscoveryRuleInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// GetDiscoveryRuleByNetwork with non-existent network ID
	_, err := storage.GetDiscoveryRuleByNetwork(context.Background(), "nonexistent")
	if err != ErrRuleNotFound {
		t.Errorf("expected ErrRuleNotFound, got %v", err)
	}

	// SaveDiscoveryRule with valid data
	rule := &model.DiscoveryRule{
		NetworkID:     network.ID,
		Enabled:       true,
		ScanType:      model.ScanTypeFull,
		IntervalHours: 24,
	}
	err = storage.SaveDiscoveryRule(context.Background(), rule)
	if err != nil {
		t.Errorf("SaveDiscoveryRule failed: %v", err)
	}

	// Verify rule was saved
	got, err := storage.GetDiscoveryRuleByNetwork(context.Background(), network.ID)
	if err != nil {
		t.Errorf("GetDiscoveryRuleByNetwork failed: %v", err)
	}
	if !got.Enabled {
		t.Error("expected rule to be enabled")
	}

	// Test GetDiscoveryRule by ID
	gotByID, err := storage.GetDiscoveryRule(context.Background(), got.ID)
	if err != nil {
		t.Errorf("GetDiscoveryRule by ID failed: %v", err)
	}
	if gotByID.NetworkID != network.ID {
		t.Errorf("expected network ID %s, got %s", network.ID, gotByID.NetworkID)
	}

	// Test DeleteDiscoveryRule
	err = storage.DeleteDiscoveryRule(context.Background(), got.ID)
	if err != nil {
		t.Errorf("DeleteDiscoveryRule failed: %v", err)
	}
	_, err = storage.GetDiscoveryRule(context.Background(), got.ID)
	if err != ErrRuleNotFound {
		t.Errorf("expected ErrRuleNotFound after delete, got %v", err)
	}
}

func TestCleanupOldDiscoveriesWithDays(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create discovered device
	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{IP: "192.168.1.1", NetworkID: network.ID})

	// Cleanup with 30 days should not remove recent devices
	if err := storage.CleanupOldDiscoveries(context.Background(), 30); err != nil {
		t.Fatalf("CleanupOldDiscoveries failed: %v", err)
	}

	devices, _ := storage.ListDiscoveredDevices(context.Background(), network.ID)
	if len(devices) != 1 {
		t.Errorf("expected 1 device (recent), got %d", len(devices))
	}
}

func TestDiscoveryScanWithAllFields(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	scan := &model.DiscoveryScan{
		NetworkID:       network.ID,
		Status:          model.ScanStatusRunning,
		ScanType:        model.ScanTypeDeep,
		TotalHosts:      254,
		ScannedHosts:    100,
		FoundHosts:      25,
		ProgressPercent: 39.4,
		ErrorMessage:    "",
	}
	if err := storage.CreateDiscoveryScan(context.Background(), scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}

	// Complete the scan
	scan.Status = model.ScanStatusCompleted
	scan.ScannedHosts = 254
	scan.FoundHosts = 50
	scan.ProgressPercent = 100.0
	if err := storage.UpdateDiscoveryScan(context.Background(), scan); err != nil {
		t.Fatalf("UpdateDiscoveryScan failed: %v", err)
	}

	got, _ := storage.GetDiscoveryScan(context.Background(), scan.ID)
	if got.Status != model.ScanStatusCompleted {
		t.Errorf("expected completed status, got %s", got.Status)
	}
	if got.FoundHosts != 50 {
		t.Errorf("expected 50 found hosts, got %d", got.FoundHosts)
	}
}

func TestDiscoveredDeviceWithServices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	device := &model.DiscoveredDevice{
		IP:        "192.168.1.50",
		NetworkID: network.ID,
		OpenPorts: []int{22, 80, 443, 3306, 5432},
		Services: []model.ServiceInfo{
			{Port: 22, Protocol: "tcp", Service: "ssh", Version: "OpenSSH 8.9"},
			{Port: 80, Protocol: "tcp", Service: "http", Version: "nginx 1.18"},
			{Port: 443, Protocol: "tcp", Service: "https", Version: "nginx 1.18"},
			{Port: 3306, Protocol: "tcp", Service: "mysql", Version: "8.0"},
			{Port: 5432, Protocol: "tcp", Service: "postgresql", Version: "14.0"},
		},
	}
	storage.CreateDiscoveredDevice(context.Background(), device)

	got, _ := storage.GetDiscoveredDevice(context.Background(), device.ID)
	if len(got.OpenPorts) != 5 {
		t.Errorf("expected 5 open ports, got %d", len(got.OpenPorts))
	}
	if len(got.Services) != 5 {
		t.Errorf("expected 5 services, got %d", len(got.Services))
	}
}

func TestListDiscoveryScansEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	scans, err := storage.ListDiscoveryScans(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("ListDiscoveryScans failed: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("expected 0 scans, got %d", len(scans))
	}
}

func TestListDiscoveryRulesEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	rules, err := storage.ListDiscoveryRules(context.Background())
	if err != nil {
		t.Fatalf("ListDiscoveryRules failed: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestDiscoveryScanWithError(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create scan with error message
	scan := &model.DiscoveryScan{
		NetworkID:    network.ID,
		Status:       model.ScanStatusFailed,
		ScanType:     model.ScanTypeFull,
		TotalHosts:   254,
		ErrorMessage: "Connection timeout",
	}
	if err := storage.CreateDiscoveryScan(context.Background(), scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}

	got, _ := storage.GetDiscoveryScan(context.Background(), scan.ID)
	if got.ErrorMessage != "Connection timeout" {
		t.Errorf("expected error message, got '%s'", got.ErrorMessage)
	}
	if got.Status != model.ScanStatusFailed {
		t.Errorf("expected failed status, got %s", got.Status)
	}
}

func TestListDiscoveryScansAllNetworks(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network1 := &model.Network{Name: "Net1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Net2", Subnet: "192.168.2.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	storage.CreateDiscoveryScan(context.Background(), &model.DiscoveryScan{NetworkID: network1.ID, Status: model.ScanStatusCompleted})
	storage.CreateDiscoveryScan(context.Background(), &model.DiscoveryScan{NetworkID: network2.ID, Status: model.ScanStatusCompleted})

	// List all scans (empty network ID)
	scans, err := storage.ListDiscoveryScans(context.Background(), "")
	if err != nil {
		t.Fatalf("ListDiscoveryScans failed: %v", err)
	}
	if len(scans) != 2 {
		t.Errorf("expected 2 scans, got %d", len(scans))
	}
}

func TestDiscoveredDeviceUpdate(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	device := &model.DiscoveredDevice{
		IP:         "192.168.1.50",
		NetworkID:  network.ID,
		Status:     "active",
		Confidence: 50,
	}
	storage.CreateDiscoveredDevice(context.Background(), device)

	// Update with new data
	device.Hostname = "updated-host"
	device.Confidence = 95
	device.OSGuess = "Linux"
	device.Vendor = "Dell"
	device.OpenPorts = []int{22, 80}
	device.Services = []model.ServiceInfo{{Port: 22, Service: "ssh"}}

	if err := storage.UpdateDiscoveredDevice(context.Background(), device); err != nil {
		t.Fatalf("UpdateDiscoveredDevice failed: %v", err)
	}

	got, _ := storage.GetDiscoveredDevice(context.Background(), device.ID)
	if got.Hostname != "updated-host" {
		t.Errorf("hostname not updated")
	}
	if got.Confidence != 95 {
		t.Errorf("confidence not updated")
	}
	if len(got.OpenPorts) != 2 {
		t.Errorf("open_ports not updated")
	}
}

func TestDiscoveryRuleWithAllFields(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	rule := &model.DiscoveryRule{
		NetworkID:     network.ID,
		Enabled:       true,
		ScanType:      model.ScanTypeDeep,
		IntervalHours: 12,
		ExcludeIPs:    "192.168.1.1,192.168.1.254",
	}
	storage.SaveDiscoveryRule(context.Background(), rule)

	got, _ := storage.GetDiscoveryRuleByNetwork(context.Background(), network.ID)
	if got.ScanType != model.ScanTypeDeep {
		t.Errorf("expected deep scan type, got %s", got.ScanType)
	}
	if got.IntervalHours != 12 {
		t.Errorf("expected 12 hour interval, got %d", got.IntervalHours)
	}
	if got.ExcludeIPs != "192.168.1.1,192.168.1.254" {
		t.Errorf("exclude_ips mismatch")
	}
}

func TestListDiscoveryRulesMultiple(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network1 := &model.Network{Name: "Net1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Net2", Subnet: "192.168.2.0/24"}
	network3 := &model.Network{Name: "Net3", Subnet: "192.168.3.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)
	storage.CreateNetwork(context.Background(), network3)

	storage.SaveDiscoveryRule(context.Background(), &model.DiscoveryRule{NetworkID: network1.ID, Enabled: true, ScanType: model.ScanTypeQuick})
	storage.SaveDiscoveryRule(context.Background(), &model.DiscoveryRule{NetworkID: network2.ID, Enabled: false, ScanType: model.ScanTypeFull})
	storage.SaveDiscoveryRule(context.Background(), &model.DiscoveryRule{NetworkID: network3.ID, Enabled: true, ScanType: model.ScanTypeDeep})

	rules, err := storage.ListDiscoveryRules(context.Background())
	if err != nil {
		t.Fatalf("ListDiscoveryRules failed: %v", err)
	}
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestDeleteDiscoveryScanAndDiscoveredDevicesByNetwork(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()
	ctx := context.Background()

	network1 := &model.Network{Name: "DeleteNet1", Subnet: "192.168.10.0/24"}
	network2 := &model.Network{Name: "DeleteNet2", Subnet: "192.168.20.0/24"}
	if err := storage.CreateNetwork(ctx, network1); err != nil {
		t.Fatalf("CreateNetwork 1 failed: %v", err)
	}
	if err := storage.CreateNetwork(ctx, network2); err != nil {
		t.Fatalf("CreateNetwork 2 failed: %v", err)
	}

	scan := &model.DiscoveryScan{
		NetworkID: network1.ID,
		Status:    model.ScanStatusCompleted,
		ScanType:  model.ScanTypeQuick,
	}
	if err := storage.CreateDiscoveryScan(ctx, scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}
	if err := storage.DeleteDiscoveryScan(ctx, scan.ID); err != nil {
		t.Fatalf("DeleteDiscoveryScan failed: %v", err)
	}
	if _, err := storage.GetDiscoveryScan(ctx, scan.ID); err != ErrScanNotFound {
		t.Fatalf("expected ErrScanNotFound, got %v", err)
	}
	if err := storage.DeleteDiscoveryScan(ctx, "missing"); err != ErrScanNotFound {
		t.Fatalf("expected ErrScanNotFound for missing scan, got %v", err)
	}

	for _, tc := range []struct {
		ip        string
		networkID string
	}{
		{"192.168.10.10", network1.ID},
		{"192.168.10.11", network1.ID},
		{"192.168.20.10", network2.ID},
	} {
		if err := storage.CreateDiscoveredDevice(ctx, &model.DiscoveredDevice{
			IP:        tc.ip,
			NetworkID: tc.networkID,
			Status:    "online",
		}); err != nil {
			t.Fatalf("CreateDiscoveredDevice failed: %v", err)
		}
	}

	if err := storage.DeleteDiscoveredDevicesByNetwork(ctx, network1.ID); err != nil {
		t.Fatalf("DeleteDiscoveredDevicesByNetwork failed: %v", err)
	}
	devices1, err := storage.ListDiscoveredDevices(ctx, network1.ID)
	if err != nil {
		t.Fatalf("ListDiscoveredDevices network1 failed: %v", err)
	}
	if len(devices1) != 0 {
		t.Fatalf("expected network1 devices to be deleted, got %d", len(devices1))
	}
	devices2, err := storage.ListDiscoveredDevices(ctx, network2.ID)
	if err != nil {
		t.Fatalf("ListDiscoveredDevices network2 failed: %v", err)
	}
	if len(devices2) != 1 {
		t.Fatalf("expected network2 devices to remain, got %d", len(devices2))
	}

	if err := storage.DeleteDiscoveredDevicesByNetwork(ctx, ""); err != nil {
		t.Fatalf("DeleteDiscoveredDevicesByNetwork all failed: %v", err)
	}
	devices2, err = storage.ListDiscoveredDevices(ctx, network2.ID)
	if err != nil {
		t.Fatalf("ListDiscoveredDevices network2 after delete all failed: %v", err)
	}
	if len(devices2) != 0 {
		t.Fatalf("expected all discovered devices to be deleted, got %d", len(devices2))
	}
}
