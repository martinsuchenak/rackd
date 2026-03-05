package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Network Pool Operations Tests
// ============================================================================

func TestPoolOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network first
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	pool := &model.NetworkPool{
		NetworkID:   network.ID,
		Name:        "DHCP Pool",
		StartIP:     "192.168.1.100",
		EndIP:       "192.168.1.200",
		Description: "Main DHCP pool",
		Tags:        []string{"dhcp", "production"},
	}

	// Create pool
	err := storage.CreateNetworkPool(context.Background(), pool)
	if err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	if pool.ID == "" {
		t.Error("pool ID should be set after creation")
	}
	if pool.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if pool.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get pool
	retrieved, err := storage.GetNetworkPool(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNetworkPool failed: %v", err)
	}

	if retrieved.Name != pool.Name {
		t.Errorf("expected name %s, got %s", pool.Name, retrieved.Name)
	}
	if retrieved.NetworkID != pool.NetworkID {
		t.Errorf("expected network_id %s, got %s", pool.NetworkID, retrieved.NetworkID)
	}
	if retrieved.StartIP != pool.StartIP {
		t.Errorf("expected start_ip %s, got %s", pool.StartIP, retrieved.StartIP)
	}
	if retrieved.EndIP != pool.EndIP {
		t.Errorf("expected end_ip %s, got %s", pool.EndIP, retrieved.EndIP)
	}
	if retrieved.Description != pool.Description {
		t.Errorf("expected description %s, got %s", pool.Description, retrieved.Description)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}
}

func TestPoolOperations_CreateWithInvalidNetwork(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	pool := &model.NetworkPool{
		NetworkID: "non-existent-network",
		Name:      "Invalid Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}

	err := storage.CreateNetworkPool(context.Background(), pool)
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestPoolOperations_CreateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateNetworkPool(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil pool")
	}
}

func TestPoolOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkPool(context.Background(), "non-existent-id")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkPool(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Original Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.150",
		Tags:      []string{"original"},
	}
	if err := storage.CreateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	originalUpdatedAt := pool.UpdatedAt

	// Update pool
	pool.Name = "Updated Pool"
	pool.StartIP = "192.168.1.50"
	pool.EndIP = "192.168.1.200"
	pool.Description = "Updated description"
	pool.Tags = []string{"updated", "production"}

	if err := storage.UpdateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("UpdateNetworkPool failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetNetworkPool(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNetworkPool failed: %v", err)
	}

	if retrieved.Name != "Updated Pool" {
		t.Errorf("expected name 'Updated Pool', got '%s'", retrieved.Name)
	}
	if retrieved.StartIP != "192.168.1.50" {
		t.Errorf("expected start_ip '192.168.1.50', got '%s'", retrieved.StartIP)
	}
	if retrieved.EndIP != "192.168.1.200" {
		t.Errorf("expected end_ip '192.168.1.200', got '%s'", retrieved.EndIP)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated_at should be updated")
	}
}

func TestPoolOperations_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	pool := &model.NetworkPool{
		ID:   "non-existent-id",
		Name: "test",
	}

	err := storage.UpdateNetworkPool(context.Background(), pool)
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_UpdateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateNetworkPool(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil pool")
	}
}

func TestPoolOperations_UpdateInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	pool := &model.NetworkPool{
		ID:   "",
		Name: "test",
	}

	err := storage.UpdateNetworkPool(context.Background(), pool)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool to delete",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
		Tags:      []string{"delete-me"},
	}
	if err := storage.CreateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	// Delete pool
	if err := storage.DeleteNetworkPool(context.Background(), pool.ID); err != nil {
		t.Fatalf("DeleteNetworkPool failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetNetworkPool(context.Background(), pool.ID)
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound after deletion, got %v", err)
	}

	// Verify cascaded deletion of tags
	var tagCount int
	storage.db.QueryRow("SELECT COUNT(*) FROM pool_tags WHERE pool_id = ?", pool.ID).Scan(&tagCount)
	if tagCount != 0 {
		t.Errorf("expected 0 tags after deletion, got %d", tagCount)
	}
}

func TestPoolOperations_DeleteWithAddresses(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create device with address in this pool
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.150", Type: "ipv4", PoolID: pool.ID},
		},
	}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete pool (should unlink addresses)
	if err := storage.DeleteNetworkPool(context.Background(), pool.ID); err != nil {
		t.Fatalf("DeleteNetworkPool failed: %v", err)
	}

	// Verify device still exists but address has no pool
	retrieved, err := storage.GetDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if len(retrieved.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(retrieved.Addresses))
	}
	if retrieved.Addresses[0].PoolID != "" {
		t.Errorf("expected empty pool_id, got '%s'", retrieved.Addresses[0].PoolID)
	}
}

func TestPoolOperations_DeleteNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteNetworkPool(context.Background(), "non-existent-id")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteNetworkPool(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create multiple pools
	pools := []string{"Pool1", "Pool2", "Pool3"}
	for i, name := range pools {
		pool := &model.NetworkPool{
			NetworkID: network.ID,
			Name:      name,
			StartIP:   "192.168.1." + string(rune('1'+i)) + "00",
			EndIP:     "192.168.1." + string(rune('1'+i)) + "50",
		}
		if err := storage.CreateNetworkPool(context.Background(), pool); err != nil {
			t.Fatalf("CreateNetworkPool failed: %v", err)
		}
	}

	// List all pools
	result, err := storage.ListNetworkPools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 pools, got %d", len(result))
	}
}

func TestPoolOperations_ListWithNetworkFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create two networks
	network1 := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	network2 := &model.Network{Name: "Network2", Subnet: "192.168.2.0/24"}
	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	// Create pools in different networks
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network1.ID, Name: "Pool1", StartIP: "192.168.1.100", EndIP: "192.168.1.200",
	})
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network1.ID, Name: "Pool2", StartIP: "192.168.1.201", EndIP: "192.168.1.250",
	})
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network2.ID, Name: "Pool3", StartIP: "192.168.2.100", EndIP: "192.168.2.200",
	})

	// Filter by network
	result, err := storage.ListNetworkPools(context.Background(), &model.NetworkPoolFilter{NetworkID: network1.ID})
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 pools in network1, got %d", len(result))
	}
}

func TestPoolOperations_ListWithTagsFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create pools with different tags
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network.ID, Name: "Pool1", StartIP: "192.168.1.100", EndIP: "192.168.1.150",
		Tags: []string{"dhcp", "production"},
	})
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network.ID, Name: "Pool2", StartIP: "192.168.1.151", EndIP: "192.168.1.200",
		Tags: []string{"dhcp", "staging"},
	})
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network.ID, Name: "Pool3", StartIP: "192.168.1.201", EndIP: "192.168.1.250",
		Tags: []string{"static"},
	})

	// Filter by single tag
	result, err := storage.ListNetworkPools(context.Background(), &model.NetworkPoolFilter{Tags: []string{"dhcp"}})
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 pools with 'dhcp' tag, got %d", len(result))
	}

	// Filter by multiple tags (AND logic)
	result, err = storage.ListNetworkPools(context.Background(), &model.NetworkPoolFilter{Tags: []string{"dhcp", "production"}})
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 pool with both tags, got %d", len(result))
	}
}

func TestPoolOperations_ListEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	result, err := storage.ListNetworkPools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}

	if result == nil {
		t.Error("result should be empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 pools, got %d", len(result))
	}
}

func TestPoolOperations_EmptyTagsNotNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool without tags
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Minimal Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	if err := storage.CreateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	// Get pool
	retrieved, err := storage.GetNetworkPool(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNetworkPool failed: %v", err)
	}

	// Verify tags is empty slice, not nil
	if retrieved.Tags == nil {
		t.Error("Tags should be empty slice, not nil")
	}
}

// ============================================================================
// GetNextAvailableIP Tests
// ============================================================================

func TestPoolOperations_GetNextAvailableIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "DHCP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.105",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Get first available IP
	ip, err := storage.GetNextAvailableIP(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNextAvailableIP failed: %v", err)
	}
	if ip != "192.168.1.100" {
		t.Errorf("expected first IP '192.168.1.100', got '%s'", ip)
	}

	// Create device using first IP
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(context.Background(), device)

	// Get next available IP (should skip used one)
	ip, err = storage.GetNextAvailableIP(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNextAvailableIP failed: %v", err)
	}
	if ip != "192.168.1.101" {
		t.Errorf("expected next IP '192.168.1.101', got '%s'", ip)
	}
}

func TestPoolOperations_GetNextAvailableIP_AllUsed(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and small pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Small Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.101",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Use all IPs
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.101", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(context.Background(), device)

	// Try to get next available IP
	_, err := storage.GetNextAvailableIP(context.Background(), pool.ID)
	if err != ErrIPNotAvailable {
		t.Errorf("expected ErrIPNotAvailable, got %v", err)
	}
}

func TestPoolOperations_GetNextAvailableIP_PoolNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNextAvailableIP(context.Background(), "non-existent-pool")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_GetNextAvailableIP_InvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNextAvailableIP(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

// ============================================================================
// ValidateIPInPool Tests
// ============================================================================

func TestPoolOperations_ValidateIPInPool(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "DHCP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.100", true},  // Start IP
		{"192.168.1.150", true},  // Middle IP
		{"192.168.1.200", true},  // End IP
		{"192.168.1.99", false},  // Just before range
		{"192.168.1.201", false}, // Just after range
		{"192.168.2.100", false}, // Different subnet
		{"10.0.0.1", false},      // Completely different
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			valid, err := storage.ValidateIPInPool(context.Background(), pool.ID, tt.ip)
			if err != nil {
				t.Fatalf("ValidateIPInPool failed: %v", err)
			}
			if valid != tt.expected {
				t.Errorf("ValidateIPInPool(%s) = %v, expected %v", tt.ip, valid, tt.expected)
			}
		})
	}
}

func TestPoolOperations_ValidateIPInPool_PoolNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.ValidateIPInPool(context.Background(), "non-existent-pool", "192.168.1.100")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_ValidateIPInPool_InvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.ValidateIPInPool(context.Background(), "", "192.168.1.100")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_ValidateIPInPool_InvalidIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "DHCP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	_, err := storage.ValidateIPInPool(context.Background(), pool.ID, "invalid-ip")
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

// ============================================================================
// GetPoolHeatmap Tests
// ============================================================================

func TestPoolOperations_GetPoolHeatmap(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and small pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Small Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.103",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create device with some addresses in the pool
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.102", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(context.Background(), device)

	// Get heatmap
	heatmap, err := storage.GetPoolHeatmap(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetPoolHeatmap failed: %v", err)
	}

	if len(heatmap) != 4 {
		t.Fatalf("expected 4 IPs in heatmap, got %d", len(heatmap))
	}

	// Verify status of each IP
	expectedStatus := map[string]string{
		"192.168.1.100": "used",
		"192.168.1.101": "available",
		"192.168.1.102": "used",
		"192.168.1.103": "available",
	}

	for _, status := range heatmap {
		expected, ok := expectedStatus[status.IP]
		if !ok {
			t.Errorf("unexpected IP in heatmap: %s", status.IP)
			continue
		}
		if status.Status != expected {
			t.Errorf("IP %s: expected status '%s', got '%s'", status.IP, expected, status.Status)
		}
		if status.Status == "used" && status.DeviceID == "" {
			t.Errorf("IP %s: used status should have device_id", status.IP)
		}
		if status.Status == "available" && status.DeviceID != "" {
			t.Errorf("IP %s: available status should not have device_id", status.IP)
		}
	}
}

func TestPoolOperations_GetPoolHeatmap_Empty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool with no used addresses
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Empty Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.102",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	heatmap, err := storage.GetPoolHeatmap(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetPoolHeatmap failed: %v", err)
	}

	if len(heatmap) != 3 {
		t.Errorf("expected 3 IPs in heatmap, got %d", len(heatmap))
	}

	for _, status := range heatmap {
		if status.Status != "available" {
			t.Errorf("IP %s: expected 'available' status, got '%s'", status.IP, status.Status)
		}
	}
}

func TestPoolOperations_GetPoolHeatmap_PoolNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetPoolHeatmap(context.Background(), "non-existent-pool")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_GetPoolHeatmap_InvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetPoolHeatmap(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_GetPoolHeatmap_SingleIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool with single IP
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Single IP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.100",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	heatmap, err := storage.GetPoolHeatmap(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetPoolHeatmap failed: %v", err)
	}

	if len(heatmap) != 1 {
		t.Errorf("expected 1 IP in heatmap, got %d", len(heatmap))
	}

	if heatmap[0].IP != "192.168.1.100" {
		t.Errorf("expected IP '192.168.1.100', got '%s'", heatmap[0].IP)
	}
}

// ============================================================================
// IP Helper Function Tests
// ============================================================================

func TestIPInRange(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		start    string
		end      string
		expected bool
	}{
		{"at start", "192.168.1.100", "192.168.1.100", "192.168.1.200", true},
		{"at end", "192.168.1.200", "192.168.1.100", "192.168.1.200", true},
		{"in middle", "192.168.1.150", "192.168.1.100", "192.168.1.200", true},
		{"before range", "192.168.1.99", "192.168.1.100", "192.168.1.200", false},
		{"after range", "192.168.1.201", "192.168.1.100", "192.168.1.200", false},
		{"different octet", "192.168.2.100", "192.168.1.100", "192.168.1.200", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIPv4(tt.ip)
			start := parseIPv4(tt.start)
			end := parseIPv4(tt.end)
			result := ipInRange(ip, start, end)
			if result != tt.expected {
				t.Errorf("ipInRange(%s, %s, %s) = %v, expected %v",
					tt.ip, tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		endIP      string
		expectedIP string
		expectedOK bool
	}{
		{"simple increment", "192.168.1.100", "192.168.1.200", "192.168.1.101", true},
		{"octet rollover", "192.168.1.255", "192.168.2.200", "192.168.2.0", true},
		{"at end", "192.168.1.200", "192.168.1.200", "192.168.1.201", false},
		{"past end", "192.168.1.201", "192.168.1.200", "192.168.1.202", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIPv4(tt.ip)
			endIP := parseIPv4(tt.endIP)
			ok := incrementIP(ip, endIP)
			if ok != tt.expectedOK {
				t.Errorf("incrementIP() returned %v, expected %v", ok, tt.expectedOK)
			}
			resultIP := fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
			if resultIP != tt.expectedIP {
				t.Errorf("IP after increment = %s, expected %s", resultIP, tt.expectedIP)
			}
		})
	}
}

// Helper to parse IPv4 for testing
func parseIPv4(s string) []byte {
	ip := make([]byte, 4)
	var a, b, c, d int
	fmt.Sscanf(s, "%d.%d.%d.%d", &a, &b, &c, &d)
	ip[0] = byte(a)
	ip[1] = byte(b)
	ip[2] = byte(c)
	ip[3] = byte(d)
	return ip
}

func TestPoolOperations_LargeRange(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool with larger range
	network := &model.Network{Name: "Network1", Subnet: "10.0.0.0/16"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Large Pool",
		StartIP:   "10.0.1.0",
		EndIP:     "10.0.1.255",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Get first available IP
	ip, err := storage.GetNextAvailableIP(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNextAvailableIP failed: %v", err)
	}
	if ip != "10.0.1.0" {
		t.Errorf("expected '10.0.1.0', got '%s'", ip)
	}
}

func TestNetworkPoolUpdateTags(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
		Tags:      []string{"original"},
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Update with new tags
	pool.Tags = []string{"updated", "new-tag", "another"}
	if err := storage.UpdateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("UpdateNetworkPool failed: %v", err)
	}

	got, _ := storage.GetNetworkPool(context.Background(), pool.ID)
	if len(got.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(got.Tags))
	}
}

func TestPoolUpdateClearTags(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
		Tags:      []string{"tag1", "tag2"},
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Update to clear tags
	pool.Tags = []string{}
	if err := storage.UpdateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("UpdateNetworkPool failed: %v", err)
	}

	got, _ := storage.GetNetworkPool(context.Background(), pool.ID)
	if len(got.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(got.Tags))
	}
}

func TestGetNextAvailableIPWithGaps(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.110",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Use IPs with gaps: 100, 102, 104
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.102", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.104", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(context.Background(), device)

	// Should return first gap: 101
	ip, err := storage.GetNextAvailableIP(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetNextAvailableIP failed: %v", err)
	}
	if ip != "192.168.1.101" {
		t.Errorf("expected '192.168.1.101', got '%s'", ip)
	}
}

func TestPoolHeatmapWithDeviceInfo(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.102",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.101", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(context.Background(), device)

	heatmap, err := storage.GetPoolHeatmap(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("GetPoolHeatmap failed: %v", err)
	}

	// Find the used IP and verify device info
	for _, status := range heatmap {
		if status.IP == "192.168.1.101" {
			if status.Status != "used" {
				t.Errorf("expected 'used' status for 192.168.1.101")
			}
			if status.DeviceID != device.ID {
				t.Errorf("expected device_id %s, got %s", device.ID, status.DeviceID)
			}
		}
	}
}

func TestListNetworkPoolsWithCombinedFilters(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network.ID, Name: "DHCP-Pool", StartIP: "192.168.1.100", EndIP: "192.168.1.150",
		Tags: []string{"dhcp", "production"},
	})
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network.ID, Name: "Static-Pool", StartIP: "192.168.1.151", EndIP: "192.168.1.200",
		Tags: []string{"static", "production"},
	})
	storage.CreateNetworkPool(context.Background(), &model.NetworkPool{
		NetworkID: network.ID, Name: "DHCP-Reserved", StartIP: "192.168.1.201", EndIP: "192.168.1.250",
		Tags: []string{"dhcp", "reserved"},
	})

	// Filter by network and tags
	result, err := storage.ListNetworkPools(context.Background(), &model.NetworkPoolFilter{NetworkID: network.ID, Tags: []string{"dhcp"}})
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 pools with 'dhcp' tag, got %d", len(result))
	}
}
