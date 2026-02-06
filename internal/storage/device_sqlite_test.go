package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Device Operations Tests
// ============================================================================

func TestDeviceOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{
		Name:        "test-server",
		Description: "Test server description",
		MakeModel:   "Dell PowerEdge R740",
		OS:          "Ubuntu 22.04",
		Username:    "admin",
		Location:    "Rack 1, Unit 5",
		Tags:        []string{"production", "web"},
		Addresses: []model.Address{
			{IP: "192.168.1.100", Port: intPtr(22), Type: "ipv4", Label: "primary"},
			{IP: "10.0.0.50", Type: "ipv4", Label: "management"},
		},
		Domains: []string{"server1.example.com", "www.example.com"},
	}

	// Create device
	err := storage.CreateDevice(context.Background(), device)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	if device.ID == "" {
		t.Error("device ID should be set after creation")
	}
	if device.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if device.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get device
	retrieved, err := storage.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	// Verify fields
	if retrieved.Name != device.Name {
		t.Errorf("expected name %s, got %s", device.Name, retrieved.Name)
	}
	if retrieved.Description != device.Description {
		t.Errorf("expected description %s, got %s", device.Description, retrieved.Description)
	}
	if retrieved.MakeModel != device.MakeModel {
		t.Errorf("expected make_model %s, got %s", device.MakeModel, retrieved.MakeModel)
	}
	if retrieved.OS != device.OS {
		t.Errorf("expected OS %s, got %s", device.OS, retrieved.OS)
	}

	// Verify tags
	if len(retrieved.Tags) != len(device.Tags) {
		t.Errorf("expected %d tags, got %d", len(device.Tags), len(retrieved.Tags))
	}

	// Verify addresses
	if len(retrieved.Addresses) != len(device.Addresses) {
		t.Errorf("expected %d addresses, got %d", len(device.Addresses), len(retrieved.Addresses))
	}

	// Verify domains
	if len(retrieved.Domains) != len(device.Domains) {
		t.Errorf("expected %d domains, got %d", len(device.Domains), len(retrieved.Domains))
	}
}

func TestDeviceOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDevice("non-existent-id")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDevice("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestDeviceOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device
	device := &model.Device{
		Name:      "original-name",
		Tags:      []string{"tag1"},
		Domains:   []string{"original.com"},
		Addresses: []model.Address{{IP: "192.168.1.1", Type: "ipv4"}},
	}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	originalUpdatedAt := device.UpdatedAt

	// Update device
	device.Name = "updated-name"
	device.Description = "updated description"
	device.Tags = []string{"tag2", "tag3"}
	device.Domains = []string{"updated.com"}
	device.Addresses = []model.Address{
		{IP: "192.168.1.2", Type: "ipv4", Label: "new-address"},
	}

	if err := storage.UpdateDevice(context.Background(), device); err != nil {
		t.Fatalf("UpdateDevice failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	if retrieved.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got '%s'", retrieved.Name)
	}
	if retrieved.Description != "updated description" {
		t.Errorf("expected description 'updated description', got '%s'", retrieved.Description)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}
	if len(retrieved.Domains) != 1 {
		t.Errorf("expected 1 domain, got %d", len(retrieved.Domains))
	}
	if len(retrieved.Addresses) != 1 {
		t.Errorf("expected 1 address, got %d", len(retrieved.Addresses))
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated_at should be updated")
	}
}

func TestDeviceOperations_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{
		ID:   "non-existent-id",
		Name: "test",
	}

	err := storage.UpdateDevice(context.Background(), device)
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device
	device := &model.Device{
		Name:      "to-delete",
		Tags:      []string{"tag1"},
		Domains:   []string{"delete.com"},
		Addresses: []model.Address{{IP: "192.168.1.1", Type: "ipv4"}},
	}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete device
	if err := storage.DeleteDevice(context.Background(), device.ID); err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetDevice(device.ID)
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound after deletion, got %v", err)
	}

	// Verify cascaded deletion of tags
	var tagCount int
	storage.db.QueryRow("SELECT COUNT(*) FROM tags WHERE device_id = ?", device.ID).Scan(&tagCount)
	if tagCount != 0 {
		t.Errorf("expected 0 tags after deletion, got %d", tagCount)
	}

	// Verify cascaded deletion of addresses
	var addrCount int
	storage.db.QueryRow("SELECT COUNT(*) FROM addresses WHERE device_id = ?", device.ID).Scan(&addrCount)
	if addrCount != 0 {
		t.Errorf("expected 0 addresses after deletion, got %d", addrCount)
	}
}

func TestDeviceOperations_DeleteNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteDevice(context.Background(), "non-existent-id")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteDevice(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestDeviceOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple devices
	devices := []string{"server1", "server2", "server3"}
	for _, name := range devices {
		device := &model.Device{Name: name}
		if err := storage.CreateDevice(context.Background(), device); err != nil {
			t.Fatalf("CreateDevice failed: %v", err)
		}
	}

	// List all devices
	result, err := storage.ListDevices(nil)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 devices, got %d", len(result))
	}
}

func TestDeviceOperations_ListWithTagsFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with different tags
	device1 := &model.Device{Name: "server1", Tags: []string{"production", "web"}}
	device2 := &model.Device{Name: "server2", Tags: []string{"production", "db"}}
	device3 := &model.Device{Name: "server3", Tags: []string{"staging"}}

	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)
	storage.CreateDevice(context.Background(), device3)

	// Filter by single tag
	result, err := storage.ListDevices(&model.DeviceFilter{Tags: []string{"production"}})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 devices with 'production' tag, got %d", len(result))
	}

	// Filter by multiple tags (AND logic)
	result, err = storage.ListDevices(&model.DeviceFilter{Tags: []string{"production", "web"}})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 device with both tags, got %d", len(result))
	}
}

func TestDeviceOperations_ListWithDatacenterFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "DC1"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create devices
	device1 := &model.Device{Name: "server1", DatacenterID: dc.ID}
	device2 := &model.Device{Name: "server2", DatacenterID: dc.ID}
	device3 := &model.Device{Name: "server3"}

	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)
	storage.CreateDevice(context.Background(), device3)

	// Filter by datacenter
	result, err := storage.ListDevices(&model.DeviceFilter{DatacenterID: dc.ID})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 devices in datacenter, got %d", len(result))
	}
}

func TestDeviceOperations_Search(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices
	device1 := &model.Device{
		Name:        "web-server",
		Description: "Main web server",
		Tags:        []string{"production"},
		Addresses:   []model.Address{{IP: "192.168.1.100", Type: "ipv4"}},
		Domains:     []string{"web.example.com"},
	}
	device2 := &model.Device{
		Name:        "db-server",
		Description: "Database server",
		Tags:        []string{"production"},
		Addresses:   []model.Address{{IP: "192.168.1.200", Type: "ipv4"}},
	}

	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)

	tests := []struct {
		query    string
		expected int
	}{
		{"web", 1},         // Match name
		{"Database", 1},    // Match description
		{"192.168.1", 2},   // Match IP addresses
		{"production", 2},  // Match tags
		{"example.com", 1}, // Match domains
		{"nonexistent", 0}, // No match
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result, err := storage.SearchDevices(tt.query)
			if err != nil {
				t.Fatalf("SearchDevices failed: %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("expected %d results for query '%s', got %d", tt.expected, tt.query, len(result))
			}
		})
	}
}

func TestDeviceOperations_SearchEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create a device
	device := &model.Device{Name: "test"}
	storage.CreateDevice(context.Background(), device)

	// Empty search returns all
	result, err := storage.SearchDevices("")
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 device for empty search, got %d", len(result))
	}
}

func TestDeviceOperations_EmptyArraysNotNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device with no tags, addresses, or domains
	device := &model.Device{Name: "minimal"}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Get device
	retrieved, err := storage.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	// Verify arrays are empty but not nil (for JSON serialization)
	if retrieved.Tags == nil {
		t.Error("Tags should be empty slice, not nil")
	}
	if retrieved.Addresses == nil {
		t.Error("Addresses should be empty slice, not nil")
	}
	if retrieved.Domains == nil {
		t.Error("Domains should be empty slice, not nil")
	}
}

func TestCreateDeviceNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateDevice(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil device")
	}
}

func TestUpdateDeviceNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateDevice(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil device")
	}
}

func TestUpdateDeviceInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{ID: "", Name: "test"}
	err := storage.UpdateDevice(context.Background(), device)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestListDevicesWithNetworkFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	// Create devices with addresses in network
	device1 := &model.Device{
		Name:      "server1",
		Addresses: []model.Address{{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID}},
	}
	device2 := &model.Device{
		Name:      "server2",
		Addresses: []model.Address{{IP: "10.0.0.1", Type: "ipv4"}},
	}
	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)

	// Filter by network
	result, err := storage.ListDevices(&model.DeviceFilter{NetworkID: network.ID})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 device in network, got %d", len(result))
	}
}

func TestDeviceWithAllFields(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter and network
	dc := &model.Datacenter{Name: "DC1", Location: "NYC"}
	storage.CreateDatacenter(context.Background(), dc)

	network := &model.Network{Name: "Net1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create device with all fields populated
	device := &model.Device{
		Name:         "full-device",
		Description:  "A fully populated device",
		MakeModel:    "Dell R740",
		OS:           "Ubuntu 22.04",
		Username:     "admin",
		Location:     "Rack 5",
		DatacenterID: dc.ID,
		Tags:         []string{"production", "web", "critical"},
		Addresses: []model.Address{
			{IP: "192.168.1.100", Port: intPtr(22), Type: "ipv4", Label: "primary", NetworkID: network.ID, PoolID: pool.ID},
			{IP: "192.168.1.101", Port: intPtr(443), Type: "ipv4", Label: "secondary", NetworkID: network.ID},
			{IP: "2001:db8::1", Type: "ipv6", Label: "ipv6"},
		},
		Domains: []string{"server.example.com", "www.example.com", "api.example.com"},
	}

	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Retrieve and verify all fields
	retrieved, err := storage.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	if retrieved.DatacenterID != dc.ID {
		t.Errorf("datacenter_id mismatch")
	}
	if len(retrieved.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(retrieved.Tags))
	}
	if len(retrieved.Addresses) != 3 {
		t.Errorf("expected 3 addresses, got %d", len(retrieved.Addresses))
	}
	if len(retrieved.Domains) != 3 {
		t.Errorf("expected 3 domains, got %d", len(retrieved.Domains))
	}

	// Verify address details
	for _, addr := range retrieved.Addresses {
		if addr.IP == "192.168.1.100" {
			if addr.NetworkID != network.ID {
				t.Errorf("address network_id mismatch")
			}
			if addr.PoolID != pool.ID {
				t.Errorf("address pool_id mismatch")
			}
		}
	}
}

func TestSearchDevicesWithSpecialCharacters(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device with special characters in name
	device := &model.Device{
		Name:        "server-01_test",
		Description: "Test with % and _ characters",
	}
	storage.CreateDevice(context.Background(), device)

	// Search should handle special SQL characters
	result, err := storage.SearchDevices("server-01")
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestDeviceUpdateClearArrays(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device with arrays
	device := &model.Device{
		Name:      "server1",
		Tags:      []string{"tag1", "tag2"},
		Domains:   []string{"domain1.com"},
		Addresses: []model.Address{{IP: "192.168.1.1", Type: "ipv4"}},
	}
	storage.CreateDevice(context.Background(), device)

	// Update to clear arrays
	device.Tags = []string{}
	device.Domains = []string{}
	device.Addresses = []model.Address{}
	if err := storage.UpdateDevice(context.Background(), device); err != nil {
		t.Fatalf("UpdateDevice failed: %v", err)
	}

	got, _ := storage.GetDevice(device.ID)
	if len(got.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(got.Tags))
	}
	if len(got.Domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(got.Domains))
	}
	if len(got.Addresses) != 0 {
		t.Errorf("expected 0 addresses, got %d", len(got.Addresses))
	}
}

func TestSearchDevicesMultipleMatches(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with overlapping search terms
	storage.CreateDevice(context.Background(), &model.Device{
		Name:        "web-server-1",
		Description: "Production web server",
		Tags:        []string{"production", "web"},
	})
	storage.CreateDevice(context.Background(), &model.Device{
		Name:        "web-server-2",
		Description: "Staging web server",
		Tags:        []string{"staging", "web"},
	})
	storage.CreateDevice(context.Background(), &model.Device{
		Name:        "db-server",
		Description: "Database server",
		Tags:        []string{"production", "database"},
	})

	// Search for "web" should match 2 devices
	result, err := storage.SearchDevices("web")
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results for 'web', got %d", len(result))
	}

	// Search for "production" should match 2 devices
	result, err = storage.SearchDevices("production")
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results for 'production', got %d", len(result))
	}
}

func TestDeviceWithMultipleAddressTypes(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{
		Name: "multi-address",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", Label: "primary"},
			{IP: "192.168.1.101", Type: "ipv4", Label: "secondary"},
			{IP: "2001:db8::1", Type: "ipv6", Label: "ipv6-primary"},
			{IP: "fe80::1", Type: "ipv6", Label: "link-local"},
		},
	}
	storage.CreateDevice(context.Background(), device)

	got, _ := storage.GetDevice(device.ID)
	if len(got.Addresses) != 4 {
		t.Errorf("expected 4 addresses, got %d", len(got.Addresses))
	}

	// Verify address types
	ipv4Count := 0
	ipv6Count := 0
	for _, addr := range got.Addresses {
		if addr.Type == "ipv4" {
			ipv4Count++
		} else if addr.Type == "ipv6" {
			ipv6Count++
		}
	}
	if ipv4Count != 2 {
		t.Errorf("expected 2 ipv4 addresses, got %d", ipv4Count)
	}
	if ipv6Count != 2 {
		t.Errorf("expected 2 ipv6 addresses, got %d", ipv6Count)
	}
}
