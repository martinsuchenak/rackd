package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Snapshot Operations Tests
// ============================================================================

func TestSnapshotOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network for reference
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	snapshot := &model.UtilizationSnapshot{
		Type:         model.SnapshotTypeNetwork,
		ResourceID:   network.ID,
		ResourceName: network.Name,
		TotalIPs:     254,
		UsedIPs:      100,
		Utilization:  39.37,
		Timestamp:    time.Now().UTC(),
	}

	// Create snapshot
	err := storage.CreateSnapshot(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot.ID == "" {
		t.Error("snapshot ID should be set after creation")
	}
	if snapshot.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}

	// List snapshots to retrieve it
	snapshots, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{ResourceID: network.ID})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}

	retrieved := snapshots[0]
	if retrieved.Type != snapshot.Type {
		t.Errorf("expected type %s, got %s", snapshot.Type, retrieved.Type)
	}
	if retrieved.ResourceID != snapshot.ResourceID {
		t.Errorf("expected resource_id %s, got %s", snapshot.ResourceID, retrieved.ResourceID)
	}
	if retrieved.TotalIPs != snapshot.TotalIPs {
		t.Errorf("expected total_ips %d, got %d", snapshot.TotalIPs, retrieved.TotalIPs)
	}
	if retrieved.UsedIPs != snapshot.UsedIPs {
		t.Errorf("expected used_ips %d, got %d", snapshot.UsedIPs, retrieved.UsedIPs)
	}
}

func TestSnapshotOperations_ListWithFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create pool
	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create snapshots for different resources
	now := time.Now().UTC()
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
		TotalIPs: 254, UsedIPs: 100, Utilization: 39.37, Timestamp: now,
	})
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypePool, ResourceID: pool.ID, ResourceName: pool.Name,
		TotalIPs: 101, UsedIPs: 50, Utilization: 49.5, Timestamp: now,
	})
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
		TotalIPs: 254, UsedIPs: 110, Utilization: 43.31, Timestamp: now.Add(-1 * time.Hour),
	})

	// Filter by type
	networkSnapshots, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{Type: model.SnapshotTypeNetwork})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(networkSnapshots) != 2 {
		t.Errorf("expected 2 network snapshots, got %d", len(networkSnapshots))
	}

	poolSnapshots, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{Type: model.SnapshotTypePool})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(poolSnapshots) != 1 {
		t.Errorf("expected 1 pool snapshot, got %d", len(poolSnapshots))
	}

	// Filter by resource ID
	filtered, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{ResourceID: network.ID})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 snapshots for network, got %d", len(filtered))
	}
}

func TestSnapshotOperations_ListWithTimeFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create snapshots at different times
	now := time.Now().UTC()
	times := []time.Duration{0, -1 * time.Hour, -2 * time.Hour, -24 * time.Hour}
	for i, offset := range times {
		storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
			Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
			TotalIPs: 254, UsedIPs: 100 + i, Utilization: 39.37, Timestamp: now.Add(offset),
		})
	}

	// Filter by After
	after := now.Add(-90 * time.Minute) // 1.5 hours ago
	filtered, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{ResourceID: network.ID, After: &after})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	// Should include snapshots at 0, -1h (2 snapshots)
	if len(filtered) != 2 {
		t.Errorf("expected 2 snapshots after filter, got %d", len(filtered))
	}

	// Filter by Before
	before := now.Add(-90 * time.Minute)
	filtered, err = storage.ListSnapshots(context.Background(), &model.SnapshotFilter{ResourceID: network.ID, Before: &before})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	// Should include snapshots at -2h, -24h (2 snapshots)
	if len(filtered) != 2 {
		t.Errorf("expected 2 snapshots before filter, got %d", len(filtered))
	}
}

func TestSnapshotOperations_ListWithLimit(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create multiple snapshots
	now := time.Now().UTC()
	for i := range 10 {
		storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
			Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
			TotalIPs: 254, UsedIPs: 100, Utilization: 39.37, Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	// List with limit
	filtered, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{Pagination: model.Pagination{Limit: 5}, ResourceID: network.ID})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(filtered) != 5 {
		t.Errorf("expected 5 snapshots with limit, got %d", len(filtered))
	}
}

func TestSnapshotOperations_GetLatestSnapshots(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create networks
	network1 := &model.Network{Name: "Network 1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Network 2", Subnet: "192.168.2.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	// Create snapshots at different times for each network
	now := time.Now().UTC()

	// Network 1: older snapshot
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network1.ID, ResourceName: network1.Name,
		TotalIPs: 254, UsedIPs: 100, Utilization: 39.37, Timestamp: now.Add(-2 * time.Hour),
	})
	// Network 1: newer snapshot
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network1.ID, ResourceName: network1.Name,
		TotalIPs: 254, UsedIPs: 110, Utilization: 43.31, Timestamp: now.Add(-1 * time.Hour),
	})

	// Network 2: only one snapshot
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network2.ID, ResourceName: network2.Name,
		TotalIPs: 254, UsedIPs: 50, Utilization: 19.69, Timestamp: now,
	})

	// Get latest snapshots
	latest, err := storage.GetLatestSnapshots(context.Background(), model.SnapshotTypeNetwork)
	if err != nil {
		t.Fatalf("GetLatestSnapshots failed: %v", err)
	}

	// Should return 2 snapshots (one per network, the most recent for each)
	if len(latest) != 2 {
		t.Errorf("expected 2 latest snapshots, got %d", len(latest))
	}

	// Verify we got the most recent for each network
	for _, snap := range latest {
		if snap.ResourceID == network1.ID && snap.UsedIPs != 110 {
			t.Errorf("expected latest snapshot for network1 to have 110 used IPs, got %d", snap.UsedIPs)
		}
		if snap.ResourceID == network2.ID && snap.UsedIPs != 50 {
			t.Errorf("expected latest snapshot for network2 to have 50 used IPs, got %d", snap.UsedIPs)
		}
	}
}

func TestSnapshotOperations_DeleteOldSnapshots(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create snapshots at different times
	now := time.Now().UTC()
	// Recent snapshot (should be kept)
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
		TotalIPs: 254, UsedIPs: 100, Utilization: 39.37, Timestamp: now.Add(-1 * time.Hour),
	})
	// Old snapshot (should be deleted)
	storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
		Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
		TotalIPs: 254, UsedIPs: 90, Utilization: 35.43, Timestamp: now.Add(-8 * 24 * time.Hour),
	})

	// Delete snapshots older than 7 days
	err := storage.DeleteOldSnapshots(context.Background(), 7)
	if err != nil {
		t.Fatalf("DeleteOldSnapshots failed: %v", err)
	}

	// Verify only recent snapshot remains
	snapshots, err := storage.ListSnapshots(context.Background(), &model.SnapshotFilter{ResourceID: network.ID})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot after cleanup, got %d", len(snapshots))
	}
	if snapshots[0].UsedIPs != 100 {
		t.Errorf("expected remaining snapshot to have 100 used IPs, got %d", snapshots[0].UsedIPs)
	}
}

func TestSnapshotOperations_GetUtilizationTrend(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create snapshots over time
	now := time.Now().UTC()
	for i := range 10 {
		storage.CreateSnapshot(context.Background(), &model.UtilizationSnapshot{
			Type: model.SnapshotTypeNetwork, ResourceID: network.ID, ResourceName: network.Name,
			TotalIPs: 254, UsedIPs: 100 + i, Utilization: float64(100+i) / 254 * 100,
			Timestamp: now.Add(time.Duration(-i) * 24 * time.Hour),
		})
	}

	// Get trend for last 5 days
	trend, err := storage.GetUtilizationTrend(context.Background(), model.SnapshotTypeNetwork, network.ID, 5)
	if err != nil {
		t.Fatalf("GetUtilizationTrend failed: %v", err)
	}

	// Should return 5 data points
	if len(trend) != 5 {
		t.Errorf("expected 5 trend points, got %d", len(trend))
	}

	// Verify ordering (should be ascending by timestamp)
	for i := 1; i < len(trend); i++ {
		if trend[i].Timestamp.Before(trend[i-1].Timestamp) {
			t.Errorf("trend points should be in ascending order by timestamp")
		}
	}
}

func TestSnapshotOperations_GetUtilizationTrendEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Get trend for non-existent resource
	trend, err := storage.GetUtilizationTrend(context.Background(), model.SnapshotTypeNetwork, "non-existent-id", 30)
	if err != nil {
		t.Fatalf("GetUtilizationTrend failed: %v", err)
	}

	if len(trend) != 0 {
		t.Errorf("expected empty trend for non-existent resource, got %d points", len(trend))
	}
}

// ============================================================================
// Dashboard Stats Tests
// ============================================================================

func TestDashboardStats_EmptyDatabase(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	stats, err := storage.GetDashboardStats(context.Background(), 7, 10)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	// Most counts should be zero (datacenters may have a default from migrations)
	if stats.TotalDevices != 0 {
		t.Errorf("expected 0 total devices, got %d", stats.TotalDevices)
	}
	if stats.TotalNetworks != 0 {
		t.Errorf("expected 0 total networks, got %d", stats.TotalNetworks)
	}
	if stats.TotalPools != 0 {
		t.Errorf("expected 0 total pools, got %d", stats.TotalPools)
	}
	if stats.DiscoveredDevices != 0 {
		t.Errorf("expected 0 discovered devices, got %d", stats.DiscoveredDevices)
	}
	if stats.StaleDevices != 0 {
		t.Errorf("expected 0 stale devices, got %d", stats.StaleDevices)
	}
}

func TestDashboardStats_WithDevices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with different statuses
	devices := []*model.Device{
		{Name: "planned1", Status: model.DeviceStatusPlanned},
		{Name: "planned2", Status: model.DeviceStatusPlanned},
		{Name: "active1", Status: model.DeviceStatusActive},
		{Name: "active2", Status: model.DeviceStatusActive},
		{Name: "active3", Status: model.DeviceStatusActive},
		{Name: "maintenance1", Status: model.DeviceStatusMaintenance},
		{Name: "decommissioned1", Status: model.DeviceStatusDecommissioned},
	}

	for _, d := range devices {
		storage.CreateDevice(context.Background(), d)
	}

	stats, err := storage.GetDashboardStats(context.Background(), 7, 10)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	if stats.TotalDevices != 7 {
		t.Errorf("expected 7 total devices, got %d", stats.TotalDevices)
	}
	if stats.DeviceStatusCounts.Planned != 2 {
		t.Errorf("expected 2 planned devices, got %d", stats.DeviceStatusCounts.Planned)
	}
	if stats.DeviceStatusCounts.Active != 3 {
		t.Errorf("expected 3 active devices, got %d", stats.DeviceStatusCounts.Active)
	}
	if stats.DeviceStatusCounts.Maintenance != 1 {
		t.Errorf("expected 1 maintenance device, got %d", stats.DeviceStatusCounts.Maintenance)
	}
	if stats.DeviceStatusCounts.Decommissioned != 1 {
		t.Errorf("expected 1 decommissioned device, got %d", stats.DeviceStatusCounts.Decommissioned)
	}
}

func TestDashboardStats_WithNetworksAndPools(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "DC1"}
	storage.CreateDatacenter(context.Background(), dc)

	// Create networks
	network1 := &model.Network{Name: "Network 1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Network 2", Subnet: "192.168.2.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	// Create pool
	pool := &model.NetworkPool{
		NetworkID: network1.ID,
		Name:      "Pool 1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	stats, err := storage.GetDashboardStats(context.Background(), 7, 10)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	if stats.TotalNetworks != 2 {
		t.Errorf("expected 2 total networks, got %d", stats.TotalNetworks)
	}
	if stats.TotalPools != 1 {
		t.Errorf("expected 1 total pool, got %d", stats.TotalPools)
	}
	// TotalDatacenters may include default from migrations, just check > 0
	if stats.TotalDatacenters < 1 {
		t.Errorf("expected at least 1 total datacenter, got %d", stats.TotalDatacenters)
	}
}

func TestDashboardStats_WithDiscoveredDevices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network 1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create discovered devices
	now := time.Now().UTC()
	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{
		IP:        "192.168.1.100",
		Hostname:  "device1",
		NetworkID: network.ID,
		FirstSeen: now,
		LastSeen:  now,
	})
	storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{
		IP:        "192.168.1.101",
		Hostname:  "device2",
		NetworkID: network.ID,
		FirstSeen: now,
		LastSeen:  now,
	})

	stats, err := storage.GetDashboardStats(context.Background(), 7, 10)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	if stats.DiscoveredDevices != 2 {
		t.Errorf("expected 2 discovered devices, got %d", stats.DiscoveredDevices)
	}
	if len(stats.RecentDiscoveries) != 2 {
		t.Errorf("expected 2 recent discoveries, got %d", len(stats.RecentDiscoveries))
	}
}

func TestDashboardStats_RecentDiscoveriesLimit(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network 1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create more discovered devices than the limit
	now := time.Now().UTC()
	for i := range 20 {
		storage.CreateDiscoveredDevice(context.Background(), &model.DiscoveredDevice{
			IP:        "192.168.1.100",
			Hostname:  "device",
			NetworkID: network.ID,
			FirstSeen: now.Add(time.Duration(-i) * time.Minute),
			LastSeen:  now,
		})
	}

	stats, err := storage.GetDashboardStats(context.Background(), 7, 5)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	if len(stats.RecentDiscoveries) > 5 {
		t.Errorf("expected at most 5 recent discoveries, got %d", len(stats.RecentDiscoveries))
	}
}

func TestDashboardStats_StaleDevices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create active device - this will be stale since no discovered device is linked
	device := &model.Device{Name: "active-device", Status: model.DeviceStatusActive}
	storage.CreateDevice(context.Background(), device)

	stats, err := storage.GetDashboardStats(context.Background(), 7, 10)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	// Device should be stale since no discovered_device record is linked
	if stats.StaleDevices != 1 {
		t.Errorf("expected 1 stale device, got %d", stats.StaleDevices)
	}
	if len(stats.StaleDeviceList) < 1 {
		t.Errorf("expected at least 1 stale device in list, got %d", len(stats.StaleDeviceList))
	}
}

func TestDashboardStats_NetworkUtilization(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network 1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create pool with addresses
	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool 1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create device with address in the network
	storage.CreateDevice(context.Background(), &model.Device{
		Name: "device1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID, PoolID: pool.ID},
		},
	})

	stats, err := storage.GetDashboardStats(context.Background(), 7, 10)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	// Should have network utilization data
	if len(stats.NetworkUtilization) != 1 {
		t.Errorf("expected 1 network utilization entry, got %d", len(stats.NetworkUtilization))
	}
	if stats.NetworkUtilization[0].NetworkName != network.Name {
		t.Errorf("expected network name '%s', got '%s'", network.Name, stats.NetworkUtilization[0].NetworkName)
	}
}
