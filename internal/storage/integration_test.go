//go:build !short

package storage

import (
	"sync"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Integration tests for storage layer
// Skip with: go test -short

func TestDeviceLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	// Create datacenter first (for device association)
	dc := &model.Datacenter{Name: "Test DC", Location: "Test Location"}
	if err := store.CreateDatacenter(dc); err != nil {
		t.Fatalf("failed to create datacenter: %v", err)
	}

	// Create network (for device address)
	network := &model.Network{Name: "Test Network", Subnet: "10.0.0.0/24", DatacenterID: dc.ID}
	if err := store.CreateNetwork(network); err != nil {
		t.Fatalf("failed to create network: %v", err)
	}

	// 1. CREATE
	device := &model.Device{
		Name:         "lifecycle-test-device",
		MakeModel:    "Dell PowerEdge",
		DatacenterID: dc.ID,
		Tags:         []string{"test", "integration"},
		Domains:      []string{"test.local"},
		Addresses: []model.Address{
			{IP: "10.0.0.10", Type: "ipv4", NetworkID: network.ID},
		},
	}
	if err := store.CreateDevice(device); err != nil {
		t.Fatalf("CREATE failed: %v", err)
	}
	if device.ID == "" {
		t.Fatal("device ID should be set after create")
	}
	deviceID := device.ID

	// 2. READ
	retrieved, err := store.GetDevice(deviceID)
	if err != nil {
		t.Fatalf("READ failed: %v", err)
	}
	if retrieved.Name != "lifecycle-test-device" {
		t.Errorf("expected name 'lifecycle-test-device', got '%s'", retrieved.Name)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}
	if len(retrieved.Addresses) != 1 {
		t.Errorf("expected 1 address, got %d", len(retrieved.Addresses))
	}

	// 3. UPDATE
	retrieved.Name = "updated-device"
	retrieved.Tags = []string{"updated"}
	retrieved.Addresses = append(retrieved.Addresses, model.Address{IP: "10.0.0.11", Type: "ipv4", NetworkID: network.ID})
	if err := store.UpdateDevice(retrieved); err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}

	// Verify update
	updated, err := store.GetDevice(deviceID)
	if err != nil {
		t.Fatalf("READ after UPDATE failed: %v", err)
	}
	if updated.Name != "updated-device" {
		t.Errorf("expected name 'updated-device', got '%s'", updated.Name)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "updated" {
		t.Errorf("tags not updated correctly: %v", updated.Tags)
	}
	if len(updated.Addresses) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(updated.Addresses))
	}

	// 4. DELETE
	if err := store.DeleteDevice(deviceID); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetDevice(deviceID)
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound after delete, got: %v", err)
	}
}

func TestNetworkPoolLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	// Create network first
	network := &model.Network{Name: "Pool Test Network", Subnet: "192.168.1.0/24"}
	if err := store.CreateNetwork(network); err != nil {
		t.Fatalf("failed to create network: %v", err)
	}

	// 1. CREATE pool
	pool := &model.NetworkPool{
		Name:      "Test Pool",
		NetworkID: network.ID,
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.110",
		Tags:      []string{"dhcp"},
	}
	if err := store.CreateNetworkPool(pool); err != nil {
		t.Fatalf("CREATE pool failed: %v", err)
	}
	poolID := pool.ID

	// 2. READ pool
	retrieved, err := store.GetNetworkPool(poolID)
	if err != nil {
		t.Fatalf("READ pool failed: %v", err)
	}
	if retrieved.Name != "Test Pool" {
		t.Errorf("expected name 'Test Pool', got '%s'", retrieved.Name)
	}

	// 3. Get next available IP
	ip, err := store.GetNextAvailableIP(poolID)
	if err != nil {
		t.Fatalf("GetNextAvailableIP failed: %v", err)
	}
	if ip != "192.168.1.100" {
		t.Errorf("expected first IP '192.168.1.100', got '%s'", ip)
	}

	// 4. Validate IP in pool
	valid, err := store.ValidateIPInPool(poolID, "192.168.1.105")
	if err != nil {
		t.Fatalf("ValidateIPInPool failed: %v", err)
	}
	if !valid {
		t.Error("192.168.1.105 should be valid in pool")
	}

	valid, err = store.ValidateIPInPool(poolID, "192.168.1.200")
	if err != nil {
		t.Fatalf("ValidateIPInPool failed: %v", err)
	}
	if valid {
		t.Error("192.168.1.200 should not be valid in pool")
	}

	// 5. UPDATE pool
	retrieved.Name = "Updated Pool"
	if err := store.UpdateNetworkPool(retrieved); err != nil {
		t.Fatalf("UPDATE pool failed: %v", err)
	}

	// 6. DELETE pool
	if err := store.DeleteNetworkPool(poolID); err != nil {
		t.Fatalf("DELETE pool failed: %v", err)
	}

	_, err = store.GetNetworkPool(poolID)
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound after delete, got: %v", err)
	}
}

func TestDiscoveryLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	// Create network
	network := &model.Network{Name: "Discovery Network", Subnet: "10.10.0.0/24"}
	if err := store.CreateNetwork(network); err != nil {
		t.Fatalf("failed to create network: %v", err)
	}

	// 1. Create discovery scan
	scan := &model.DiscoveryScan{
		NetworkID: network.ID,
		ScanType:  model.ScanTypeQuick,
		Status:    model.ScanStatusPending,
	}
	if err := store.CreateDiscoveryScan(scan); err != nil {
		t.Fatalf("CREATE scan failed: %v", err)
	}

	// 2. Update scan status
	now := time.Now()
	scan.Status = model.ScanStatusRunning
	scan.StartedAt = &now
	if err := store.UpdateDiscoveryScan(scan); err != nil {
		t.Fatalf("UPDATE scan failed: %v", err)
	}

	// 3. Create discovered device
	discovered := &model.DiscoveredDevice{
		NetworkID: network.ID,
		IP:        "10.10.0.50",
		Hostname:  "discovered-host",
		Status:    "active",
		OpenPorts: []int{22, 80},
		Services:  []model.ServiceInfo{{Port: 22, Protocol: "tcp", Service: "ssh"}},
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
	}
	if err := store.CreateDiscoveredDevice(discovered); err != nil {
		t.Fatalf("CREATE discovered device failed: %v", err)
	}

	// 4. Get by IP
	byIP, err := store.GetDiscoveredDeviceByIP(network.ID, "10.10.0.50")
	if err != nil {
		t.Fatalf("GetDiscoveredDeviceByIP failed: %v", err)
	}
	if byIP.Hostname != "discovered-host" {
		t.Errorf("expected hostname 'discovered-host', got '%s'", byIP.Hostname)
	}

	// 5. Promote to device
	device := &model.Device{Name: "promoted-device", MakeModel: "Unknown"}
	if err := store.CreateDevice(device); err != nil {
		t.Fatalf("CREATE device for promotion failed: %v", err)
	}
	if err := store.PromoteDiscoveredDevice(discovered.ID, device.ID); err != nil {
		t.Fatalf("PromoteDiscoveredDevice failed: %v", err)
	}

	// Verify promotion
	promoted, err := store.GetDiscoveredDevice(discovered.ID)
	if err != nil {
		t.Fatalf("GET after promotion failed: %v", err)
	}
	if promoted.PromotedToDeviceID != device.ID {
		t.Errorf("expected promoted_to_device_id '%s', got '%s'", device.ID, promoted.PromotedToDeviceID)
	}

	// 6. Complete scan
	completedAt := time.Now()
	scan.Status = model.ScanStatusCompleted
	scan.CompletedAt = &completedAt
	scan.FoundHosts = 1
	if err := store.UpdateDiscoveryScan(scan); err != nil {
		t.Fatalf("UPDATE scan completion failed: %v", err)
	}
}

func TestRelationshipLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	// Create parent and child devices
	parent := &model.Device{Name: "parent-device", MakeModel: "Rack"}
	if err := store.CreateDevice(parent); err != nil {
		t.Fatalf("CREATE parent failed: %v", err)
	}

	child := &model.Device{Name: "child-device", MakeModel: "Server"}
	if err := store.CreateDevice(child); err != nil {
		t.Fatalf("CREATE child failed: %v", err)
	}

	// 1. Add relationship
	if err := store.AddRelationship(parent.ID, child.ID, model.RelationshipContains, ""); err != nil {
		t.Fatalf("AddRelationship failed: %v", err)
	}

	// 2. Get relationships
	rels, err := store.GetRelationships(parent.ID)
	if err != nil {
		t.Fatalf("GetRelationships failed: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rels))
	}

	// 3. Get related devices
	related, err := store.GetRelatedDevices(parent.ID, model.RelationshipContains)
	if err != nil {
		t.Fatalf("GetRelatedDevices failed: %v", err)
	}
	if len(related) != 1 || related[0].ID != child.ID {
		t.Errorf("expected child device in related, got %v", related)
	}

	// 4. Remove relationship
	if err := store.RemoveRelationship(parent.ID, child.ID, model.RelationshipContains); err != nil {
		t.Fatalf("RemoveRelationship failed: %v", err)
	}

	// Verify removal
	rels, err = store.GetRelationships(parent.ID)
	if err != nil {
		t.Fatalf("GetRelationships after removal failed: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships after removal, got %d", len(rels))
	}
}

func TestMigrationOnFreshDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create fresh storage (runs migrations)
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer store.Close()

	// Verify all tables exist
	tables := []string{
		"devices", "addresses", "tags", "domains",
		"datacenters", "networks", "network_pools", "pool_tags",
		"device_relationships",
		"discovered_devices", "discovery_scans", "discovery_rules",
		"schema_migrations",
	}

	for _, table := range tables {
		var name string
		err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}

	// Verify we can perform basic operations
	device := &model.Device{Name: "migration-test", MakeModel: "Test"}
	if err := store.CreateDevice(device); err != nil {
		t.Errorf("failed to create device after migration: %v", err)
	}
}

func TestConcurrentDeviceAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	// Create initial device
	device := &model.Device{Name: "concurrent-test", MakeModel: "Test", Tags: []string{}}
	if err := store.CreateDevice(device); err != nil {
		t.Fatalf("CREATE failed: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.GetDevice(device.ID)
			if err != nil {
				errors <- err
			}
		}()
	}

	// Concurrent list operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.ListDevices(nil)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Concurrent device creation
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			device := &model.Device{
				Name:      "concurrent-device-" + string(rune('A'+n)),
				MakeModel: "Test",
				Tags:      []string{},
			}
			if err := store.CreateDevice(device); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify all devices created
	devices, err := store.ListDevices(nil)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(devices) != 10 {
		t.Errorf("expected 10 devices, got %d", len(devices))
	}
}

func TestConcurrentPoolOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store := newTestStorage(t)

	// Create network and pool
	network := &model.Network{Name: "Concurrent Pool Network", Subnet: "172.16.0.0/24"}
	if err := store.CreateNetwork(network); err != nil {
		t.Fatalf("failed to create network: %v", err)
	}

	pool := &model.NetworkPool{
		Name:      "Concurrent Pool",
		NetworkID: network.ID,
		StartIP:   "172.16.0.1",
		EndIP:     "172.16.0.100",
		Tags:      []string{},
	}
	if err := store.CreateNetworkPool(pool); err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	var wg sync.WaitGroup
	ips := make(chan string, 20)
	errors := make(chan error, 20)

	// Concurrent GetNextAvailableIP calls
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ip, err := store.GetNextAvailableIP(pool.ID)
			if err != nil {
				errors <- err
				return
			}
			ips <- ip
		}()
	}

	wg.Wait()
	close(ips)
	close(errors)

	for err := range errors {
		t.Errorf("concurrent pool operation error: %v", err)
	}

	// All should return the same first available IP (no actual allocation)
	var count int
	for range ips {
		count++
	}
	if count != 20 {
		t.Errorf("expected 20 IP results, got %d", count)
	}
}
