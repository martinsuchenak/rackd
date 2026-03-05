package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Network Operations Tests
// ============================================================================

func TestNetworkOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{
		Name:        "Production Network",
		Subnet:      "192.168.1.0/24",
		VLANID:      100,
		Description: "Main production network",
	}

	// Create network
	err := storage.CreateNetwork(context.Background(), network)
	if err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	if network.ID == "" {
		t.Error("network ID should be set after creation")
	}
	if network.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if network.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get network
	retrieved, err := storage.GetNetwork(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetwork failed: %v", err)
	}

	if retrieved.Name != network.Name {
		t.Errorf("expected name %s, got %s", network.Name, retrieved.Name)
	}
	if retrieved.Subnet != network.Subnet {
		t.Errorf("expected subnet %s, got %s", network.Subnet, retrieved.Subnet)
	}
	if retrieved.VLANID != network.VLANID {
		t.Errorf("expected VLAN ID %d, got %d", network.VLANID, retrieved.VLANID)
	}
	if retrieved.Description != network.Description {
		t.Errorf("expected description %s, got %s", network.Description, retrieved.Description)
	}
}

func TestNetworkOperations_CreateWithDatacenter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter first
	dc := &model.Datacenter{Name: "DC1"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	network := &model.Network{
		Name:         "DC Network",
		Subnet:       "10.0.0.0/16",
		DatacenterID: dc.ID,
	}

	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	retrieved, err := storage.GetNetwork(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetwork failed: %v", err)
	}

	if retrieved.DatacenterID != dc.ID {
		t.Errorf("expected datacenter_id %s, got %s", dc.ID, retrieved.DatacenterID)
	}
}

func TestNetworkOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetwork(context.Background(), "non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetwork(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{
		Name:   "Original Network",
		Subnet: "192.168.1.0/24",
		VLANID: 100,
	}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	originalUpdatedAt := network.UpdatedAt

	// Update network
	network.Name = "Updated Network"
	network.Subnet = "10.0.0.0/16"
	network.VLANID = 200
	network.Description = "Updated description"

	if err := storage.UpdateNetwork(context.Background(), network); err != nil {
		t.Fatalf("UpdateNetwork failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetNetwork(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetwork failed: %v", err)
	}

	if retrieved.Name != "Updated Network" {
		t.Errorf("expected name 'Updated Network', got '%s'", retrieved.Name)
	}
	if retrieved.Subnet != "10.0.0.0/16" {
		t.Errorf("expected subnet '10.0.0.0/16', got '%s'", retrieved.Subnet)
	}
	if retrieved.VLANID != 200 {
		t.Errorf("expected VLAN ID 200, got %d", retrieved.VLANID)
	}
	if retrieved.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", retrieved.Description)
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated_at should be updated")
	}
}

func TestNetworkOperations_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{
		ID:   "non-existent-id",
		Name: "test",
	}

	err := storage.UpdateNetwork(context.Background(), network)
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_UpdateInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{
		ID:   "",
		Name: "test",
	}

	err := storage.UpdateNetwork(context.Background(), network)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network-to-delete", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Delete network
	if err := storage.DeleteNetwork(context.Background(), network.ID); err != nil {
		t.Fatalf("DeleteNetwork failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetNetwork(context.Background(), network.ID)
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound after deletion, got %v", err)
	}
}

func TestNetworkOperations_DeleteWithAddresses(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create device with address in this network
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID},
		},
	}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete network (should unlink addresses)
	if err := storage.DeleteNetwork(context.Background(), network.ID); err != nil {
		t.Fatalf("DeleteNetwork failed: %v", err)
	}

	// Verify device still exists but address has no network
	retrieved, err := storage.GetDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if len(retrieved.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(retrieved.Addresses))
	}
	if retrieved.Addresses[0].NetworkID != "" {
		t.Errorf("expected empty network_id, got '%s'", retrieved.Addresses[0].NetworkID)
	}
}

func TestNetworkOperations_DeleteNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteNetwork(context.Background(), "non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteNetwork(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple networks
	networks := []struct {
		name   string
		subnet string
	}{
		{"Network1", "192.168.1.0/24"},
		{"Network2", "192.168.2.0/24"},
		{"Network3", "10.0.0.0/16"},
	}
	for _, n := range networks {
		network := &model.Network{Name: n.name, Subnet: n.subnet}
		if err := storage.CreateNetwork(context.Background(), network); err != nil {
			t.Fatalf("CreateNetwork failed: %v", err)
		}
	}

	// List all networks
	result, err := storage.ListNetworks(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListNetworks failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 networks, got %d", len(result))
	}
}

func TestNetworkOperations_ListWithNameFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create networks
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Production-1", Subnet: "192.168.1.0/24"})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Production-2", Subnet: "192.168.2.0/24"})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Staging", Subnet: "10.0.0.0/16"})

	// Filter by name
	result, err := storage.ListNetworks(context.Background(), &model.NetworkFilter{Name: "Production"})
	if err != nil {
		t.Fatalf("ListNetworks failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 networks matching 'Production', got %d", len(result))
	}
}

func TestNetworkOperations_ListWithDatacenterFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenters
	dc1 := &model.Datacenter{Name: "DC1"}
	dc2 := &model.Datacenter{Name: "DC2"}
	storage.CreateDatacenter(context.Background(), dc1)
	storage.CreateDatacenter(context.Background(), dc2)

	// Create networks
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network1", Subnet: "192.168.1.0/24", DatacenterID: dc1.ID})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network2", Subnet: "192.168.2.0/24", DatacenterID: dc1.ID})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network3", Subnet: "10.0.0.0/16", DatacenterID: dc2.ID})

	// Filter by datacenter
	result, err := storage.ListNetworks(context.Background(), &model.NetworkFilter{DatacenterID: dc1.ID})
	if err != nil {
		t.Fatalf("ListNetworks failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 networks in DC1, got %d", len(result))
	}
}

func TestNetworkOperations_ListWithVLANFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create networks
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network1", Subnet: "192.168.1.0/24", VLANID: 100})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network2", Subnet: "192.168.2.0/24", VLANID: 100})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network3", Subnet: "10.0.0.0/16", VLANID: 200})

	// Filter by VLAN
	result, err := storage.ListNetworks(context.Background(), &model.NetworkFilter{VLANID: 100})
	if err != nil {
		t.Fatalf("ListNetworks failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 networks with VLAN 100, got %d", len(result))
	}
}

func TestNetworkOperations_ListEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	result, err := storage.ListNetworks(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListNetworks failed: %v", err)
	}

	if result == nil {
		t.Error("result should be empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 networks, got %d", len(result))
	}
}

func TestNetworkOperations_GetNetworkDevices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create devices with addresses in this network
	device1 := &model.Device{
		Name:      "server1",
		Addresses: []model.Address{{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID}},
	}
	device2 := &model.Device{
		Name:      "server2",
		Addresses: []model.Address{{IP: "192.168.1.101", Type: "ipv4", NetworkID: network.ID}},
	}
	device3 := &model.Device{
		Name:      "server3",
		Addresses: []model.Address{{IP: "10.0.0.1", Type: "ipv4"}}, // Different network
	}

	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)
	storage.CreateDevice(context.Background(), device3)

	// Get devices in network
	devices, err := storage.GetNetworkDevices(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetworkDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("expected 2 devices in network, got %d", len(devices))
	}
}

func TestNetworkOperations_GetNetworkDevicesNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkDevices(context.Background(), "non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkDevicesEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with no devices
	network := &model.Network{Name: "Empty-Network", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	devices, err := storage.GetNetworkDevices(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetworkDevices failed: %v", err)
	}

	if devices == nil {
		t.Error("devices should be empty slice, not nil")
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestNetworkOperations_GetNetworkDevicesInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkDevices(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkUtilization(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with /24 subnet (254 usable IPs)
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create devices with addresses in this network
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID},
			{IP: "192.168.1.101", Type: "ipv4", NetworkID: network.ID},
		},
	}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Get utilization
	util, err := storage.GetNetworkUtilization(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetworkUtilization failed: %v", err)
	}

	if util.NetworkID != network.ID {
		t.Errorf("expected network_id %s, got %s", network.ID, util.NetworkID)
	}
	// /24 has 256 - 2 = 254 usable addresses
	if util.TotalIPs != 254 {
		t.Errorf("expected 254 total IPs for /24, got %d", util.TotalIPs)
	}
	if util.UsedIPs != 2 {
		t.Errorf("expected 2 used IPs, got %d", util.UsedIPs)
	}
	if util.AvailableIPs != 252 {
		t.Errorf("expected 252 available IPs, got %d", util.AvailableIPs)
	}
	// Utilization should be 2/254 * 100 ≈ 0.787%
	if util.Utilization < 0.5 || util.Utilization > 1.0 {
		t.Errorf("expected utilization around 0.787%%, got %.2f%%", util.Utilization)
	}
}

func TestNetworkOperations_GetNetworkUtilizationNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkUtilization(context.Background(), "non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkUtilizationInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkUtilization(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkUtilizationEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with no devices
	network := &model.Network{Name: "Empty-Network", Subnet: "10.0.0.0/16"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	util, err := storage.GetNetworkUtilization(context.Background(), network.ID)
	if err != nil {
		t.Fatalf("GetNetworkUtilization failed: %v", err)
	}

	if util.UsedIPs != 0 {
		t.Errorf("expected 0 used IPs, got %d", util.UsedIPs)
	}
	if util.Utilization != 0 {
		t.Errorf("expected 0%% utilization, got %.2f%%", util.Utilization)
	}
}

func TestCalculateCIDRSize(t *testing.T) {
	tests := []struct {
		cidr     string
		expected int
		hasError bool
	}{
		{"192.168.1.0/24", 254, false}, // 256 - 2 (network + broadcast)
		{"10.0.0.0/16", 65534, false},  // 65536 - 2
		{"192.168.1.0/30", 2, false},   // 4 - 2
		{"192.168.1.0/31", 2, false},   // Point-to-point link
		{"192.168.1.1/32", 1, false},   // Single host
		{"invalid", 0, true},           // Invalid CIDR
		{"192.168.1.0", 0, true},       // Missing prefix
	}

	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			result, err := calculateCIDRSize(tt.cidr)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for CIDR %s", tt.cidr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for CIDR %s: %v", tt.cidr, err)
				return
			}
			if result != tt.expected {
				t.Errorf("calculateCIDRSize(%s) = %d, expected %d", tt.cidr, result, tt.expected)
			}
		})
	}
}

func TestNetworkOperations_CreateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateNetwork(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil network")
	}
}

func TestNetworkOperations_UpdateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateNetwork(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil network")
	}
}

func TestNetworkUtilizationInvalidSubnet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with invalid subnet
	network := &model.Network{Name: "BadNet", Subnet: "invalid-cidr"}
	storage.CreateNetwork(context.Background(), network)

	_, err := storage.GetNetworkUtilization(context.Background(), network.ID)
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestDeleteNetworkWithPools(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Delete network (should cascade to pools)
	if err := storage.DeleteNetwork(context.Background(), network.ID); err != nil {
		t.Fatalf("DeleteNetwork failed: %v", err)
	}

	// Verify pool is deleted
	_, err := storage.GetNetworkPool(context.Background(), pool.ID)
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound after network deletion, got %v", err)
	}
}

func TestNetworkWithZeroVLAN(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with VLAN 0 (untagged)
	network := &model.Network{
		Name:   "Untagged Network",
		Subnet: "192.168.1.0/24",
		VLANID: 0,
	}
	storage.CreateNetwork(context.Background(), network)

	got, _ := storage.GetNetwork(context.Background(), network.ID)
	if got.VLANID != 0 {
		t.Errorf("expected VLAN 0, got %d", got.VLANID)
	}
}

func TestCalculateCIDRSizeEdgeCases(t *testing.T) {
	tests := []struct {
		cidr     string
		expected int
		hasError bool
	}{
		{"10.0.0.0/8", 1 << 20, false},   // Large network (capped at ~1M)
		{"192.168.1.128/25", 126, false}, // /25 subnet
		{"192.168.1.192/26", 62, false},  // /26 subnet
		{"192.168.1.240/28", 14, false},  // /28 subnet
		{"192.168.1.252/30", 2, false},   // /30 subnet
	}

	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			result, err := calculateCIDRSize(tt.cidr)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for CIDR %s", tt.cidr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for CIDR %s: %v", tt.cidr, err)
				return
			}
			if result != tt.expected {
				t.Errorf("calculateCIDRSize(%s) = %d, expected %d", tt.cidr, result, tt.expected)
			}
		})
	}
}
