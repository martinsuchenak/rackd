package storage

import (
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestNewSQLiteStorage(t *testing.T) {
	// Test with in-memory database
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Verify database connection is valid
	if storage.db == nil {
		t.Error("database connection should not be nil")
	}

	// Verify we can ping the database
	if err := storage.db.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestNewSQLiteStorageWithDataDir(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()

	storage, err := NewSQLiteStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Verify database connection is valid
	if storage.db == nil {
		t.Error("database connection should not be nil")
	}

	// Verify we can ping the database
	if err := storage.db.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestSQLiteStorageRunsMigrations(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Verify migrations were run by checking for tables
	tables := []string{"devices", "networks", "datacenters"}
	for _, table := range tables {
		var name string
		err := storage.db.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name=?
		`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestSQLiteStorageClose(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}

	// Close the storage
	if err := storage.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify database is closed (ping should fail)
	if err := storage.db.Ping(); err == nil {
		t.Error("expected ping to fail after close")
	}
}

func TestNewUUID(t *testing.T) {
	// Generate multiple UUIDs and verify they're unique
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newUUID()
		if id == "" {
			t.Error("UUID should not be empty")
		}
		if seen[id] {
			t.Errorf("duplicate UUID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestFactoryFunctions(t *testing.T) {
	t.Run("NewStorage", func(t *testing.T) {
		storage, err := NewStorage(":memory:")
		if err != nil {
			t.Fatalf("NewStorage failed: %v", err)
		}

		// Verify it implements Storage interface
		var _ Storage = storage

		// Close
		if s, ok := storage.(*SQLiteStorage); ok {
			s.Close()
		}
	})

	t.Run("NewExtendedStorage", func(t *testing.T) {
		storage, err := NewExtendedStorage(":memory:")
		if err != nil {
			t.Fatalf("NewExtendedStorage failed: %v", err)
		}

		// Verify it implements ExtendedStorage interface
		var _ ExtendedStorage = storage

		// Close
		storage.Close()
	})
}

func TestSQLiteStorageImplementsInterfaces(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Verify SQLiteStorage implements all interfaces
	var _ DeviceStorage = storage
	var _ DatacenterStorage = storage
	var _ NetworkStorage = storage
	var _ NetworkPoolStorage = storage
	var _ RelationshipStorage = storage
	var _ DiscoveryStorage = storage
	var _ ExtendedStorage = storage
}

func TestWALModeEnabled(t *testing.T) {
	// Use temp directory to test file-based database
	tmpDir := t.TempDir()

	storage, err := NewSQLiteStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Check journal mode is WAL
	var journalMode string
	err = storage.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("expected journal_mode to be 'wal', got '%s'", journalMode)
	}
}

// Helper function to create a test storage
func newTestStorage(t *testing.T) *SQLiteStorage {
	t.Helper()
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create test storage: %v", err)
	}
	return storage
}

// ============================================================================
// Device Operations Tests (P2-004)
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
			{IP: "192.168.1.100", Port: 22, Type: "ipv4", Label: "primary"},
			{IP: "10.0.0.50", Type: "ipv4", Label: "management"},
		},
		Domains: []string{"server1.example.com", "www.example.com"},
	}

	// Create device
	err := storage.CreateDevice(device)
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
		Name:     "original-name",
		Tags:     []string{"tag1"},
		Domains:  []string{"original.com"},
		Addresses: []model.Address{{IP: "192.168.1.1", Type: "ipv4"}},
	}
	if err := storage.CreateDevice(device); err != nil {
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

	if err := storage.UpdateDevice(device); err != nil {
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

	err := storage.UpdateDevice(device)
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device
	device := &model.Device{
		Name:     "to-delete",
		Tags:     []string{"tag1"},
		Domains:  []string{"delete.com"},
		Addresses: []model.Address{{IP: "192.168.1.1", Type: "ipv4"}},
	}
	if err := storage.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete device
	if err := storage.DeleteDevice(device.ID); err != nil {
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

	err := storage.DeleteDevice("non-existent-id")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDeviceOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteDevice("")
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
		if err := storage.CreateDevice(device); err != nil {
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

	storage.CreateDevice(device1)
	storage.CreateDevice(device2)
	storage.CreateDevice(device3)

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
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create devices
	device1 := &model.Device{Name: "server1", DatacenterID: dc.ID}
	device2 := &model.Device{Name: "server2", DatacenterID: dc.ID}
	device3 := &model.Device{Name: "server3"}

	storage.CreateDevice(device1)
	storage.CreateDevice(device2)
	storage.CreateDevice(device3)

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

	storage.CreateDevice(device1)
	storage.CreateDevice(device2)

	tests := []struct {
		query    string
		expected int
	}{
		{"web", 1},       // Match name
		{"Database", 1},  // Match description
		{"192.168.1", 2}, // Match IP addresses
		{"production", 2}, // Match tags
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
	storage.CreateDevice(device)

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
	if err := storage.CreateDevice(device); err != nil {
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

// ============================================================================
// Datacenter Operations Tests (P2-005)
// ============================================================================

func TestDatacenterOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	dc := &model.Datacenter{
		Name:        "DC1",
		Location:    "New York",
		Description: "Primary datacenter",
	}

	// Create datacenter
	err := storage.CreateDatacenter(dc)
	if err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	if dc.ID == "" {
		t.Error("datacenter ID should be set after creation")
	}
	if dc.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}

	// Get datacenter
	retrieved, err := storage.GetDatacenter(dc.ID)
	if err != nil {
		t.Fatalf("GetDatacenter failed: %v", err)
	}

	if retrieved.Name != dc.Name {
		t.Errorf("expected name %s, got %s", dc.Name, retrieved.Name)
	}
	if retrieved.Location != dc.Location {
		t.Errorf("expected location %s, got %s", dc.Location, retrieved.Location)
	}
	if retrieved.Description != dc.Description {
		t.Errorf("expected description %s, got %s", dc.Description, retrieved.Description)
	}
}

func TestDatacenterOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDatacenter("non-existent-id")
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound, got %v", err)
	}
}

func TestDatacenterOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDatacenter("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestDatacenterOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{
		Name:     "DC1",
		Location: "New York",
	}
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	originalUpdatedAt := dc.UpdatedAt

	// Update datacenter
	dc.Name = "DC1-Updated"
	dc.Location = "Chicago"
	dc.Description = "Updated description"

	if err := storage.UpdateDatacenter(dc); err != nil {
		t.Fatalf("UpdateDatacenter failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetDatacenter(dc.ID)
	if err != nil {
		t.Fatalf("GetDatacenter failed: %v", err)
	}

	if retrieved.Name != "DC1-Updated" {
		t.Errorf("expected name 'DC1-Updated', got '%s'", retrieved.Name)
	}
	if retrieved.Location != "Chicago" {
		t.Errorf("expected location 'Chicago', got '%s'", retrieved.Location)
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated_at should be updated")
	}
}

func TestDatacenterOperations_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	dc := &model.Datacenter{
		ID:   "non-existent-id",
		Name: "test",
	}

	err := storage.UpdateDatacenter(dc)
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound, got %v", err)
	}
}

func TestDatacenterOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "DC-to-delete"}
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Delete datacenter
	if err := storage.DeleteDatacenter(dc.ID); err != nil {
		t.Fatalf("DeleteDatacenter failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetDatacenter(dc.ID)
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound after deletion, got %v", err)
	}
}

func TestDatacenterOperations_DeleteWithDevices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "DC1"}
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create devices in datacenter
	device := &model.Device{Name: "server1", DatacenterID: dc.ID}
	if err := storage.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete datacenter (should unlink devices)
	if err := storage.DeleteDatacenter(dc.ID); err != nil {
		t.Fatalf("DeleteDatacenter failed: %v", err)
	}

	// Verify device still exists but with no datacenter
	retrieved, err := storage.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if retrieved.DatacenterID != "" {
		t.Errorf("expected empty datacenter_id, got '%s'", retrieved.DatacenterID)
	}
}

func TestDatacenterOperations_DeleteNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteDatacenter("non-existent-id")
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound, got %v", err)
	}
}

func TestDatacenterOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteDatacenter("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestDatacenterOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple datacenters
	names := []string{"DC1", "DC2", "DC3"}
	for _, name := range names {
		dc := &model.Datacenter{Name: name}
		if err := storage.CreateDatacenter(dc); err != nil {
			t.Fatalf("CreateDatacenter failed: %v", err)
		}
	}

	// List all datacenters
	result, err := storage.ListDatacenters(nil)
	if err != nil {
		t.Fatalf("ListDatacenters failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 datacenters, got %d", len(result))
	}
}

func TestDatacenterOperations_ListWithFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenters
	storage.CreateDatacenter(&model.Datacenter{Name: "NYC-DC1"})
	storage.CreateDatacenter(&model.Datacenter{Name: "NYC-DC2"})
	storage.CreateDatacenter(&model.Datacenter{Name: "LA-DC1"})

	// Filter by name
	result, err := storage.ListDatacenters(&model.DatacenterFilter{Name: "NYC"})
	if err != nil {
		t.Fatalf("ListDatacenters failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 datacenters matching 'NYC', got %d", len(result))
	}
}

func TestDatacenterOperations_ListEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	result, err := storage.ListDatacenters(nil)
	if err != nil {
		t.Fatalf("ListDatacenters failed: %v", err)
	}

	if result == nil {
		t.Error("result should be empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 datacenters, got %d", len(result))
	}
}

func TestDatacenterOperations_GetDatacenterDevices(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "DC1"}
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create devices in datacenter
	device1 := &model.Device{Name: "server1", DatacenterID: dc.ID}
	device2 := &model.Device{Name: "server2", DatacenterID: dc.ID}
	device3 := &model.Device{Name: "server3"} // Not in datacenter

	storage.CreateDevice(device1)
	storage.CreateDevice(device2)
	storage.CreateDevice(device3)

	// Get devices in datacenter
	devices, err := storage.GetDatacenterDevices(dc.ID)
	if err != nil {
		t.Fatalf("GetDatacenterDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("expected 2 devices in datacenter, got %d", len(devices))
	}
}

func TestDatacenterOperations_GetDatacenterDevicesNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDatacenterDevices("non-existent-id")
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound, got %v", err)
	}
}

func TestDatacenterOperations_GetDatacenterDevicesEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter with no devices
	dc := &model.Datacenter{Name: "Empty-DC"}
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	devices, err := storage.GetDatacenterDevices(dc.ID)
	if err != nil {
		t.Fatalf("GetDatacenterDevices failed: %v", err)
	}

	if devices == nil {
		t.Error("devices should be empty slice, not nil")
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestDatacenterOperations_GetDatacenterDevicesInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDatacenterDevices("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestNullString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"value", true},
	}

	for _, tt := range tests {
		result := nullString(tt.input)
		if result.Valid != tt.expected {
			t.Errorf("nullString(%q).Valid = %v, expected %v", tt.input, result.Valid, tt.expected)
		}
		if tt.expected && result.String != tt.input {
			t.Errorf("nullString(%q).String = %q, expected %q", tt.input, result.String, tt.input)
		}
	}
}

func TestNullInt(t *testing.T) {
	tests := []struct {
		input    int
		expected bool
	}{
		{0, false},
		{1, true},
		{-1, true},
	}

	for _, tt := range tests {
		result := nullInt(tt.input)
		if result.Valid != tt.expected {
			t.Errorf("nullInt(%d).Valid = %v, expected %v", tt.input, result.Valid, tt.expected)
		}
		if tt.expected && result.Int64 != int64(tt.input) {
			t.Errorf("nullInt(%d).Int64 = %d, expected %d", tt.input, result.Int64, tt.input)
		}
	}
}

// ============================================================================
// Network Operations Tests (P2-006)
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
	err := storage.CreateNetwork(network)
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
	retrieved, err := storage.GetNetwork(network.ID)
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
	if err := storage.CreateDatacenter(dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	network := &model.Network{
		Name:         "DC Network",
		Subnet:       "10.0.0.0/16",
		DatacenterID: dc.ID,
	}

	if err := storage.CreateNetwork(network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	retrieved, err := storage.GetNetwork(network.ID)
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

	_, err := storage.GetNetwork("non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetwork("")
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
	if err := storage.CreateNetwork(network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	originalUpdatedAt := network.UpdatedAt

	// Update network
	network.Name = "Updated Network"
	network.Subnet = "10.0.0.0/16"
	network.VLANID = 200
	network.Description = "Updated description"

	if err := storage.UpdateNetwork(network); err != nil {
		t.Fatalf("UpdateNetwork failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetNetwork(network.ID)
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

	err := storage.UpdateNetwork(network)
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

	err := storage.UpdateNetwork(network)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network-to-delete", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Delete network
	if err := storage.DeleteNetwork(network.ID); err != nil {
		t.Fatalf("DeleteNetwork failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetNetwork(network.ID)
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound after deletion, got %v", err)
	}
}

func TestNetworkOperations_DeleteWithAddresses(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create device with address in this network
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID},
		},
	}
	if err := storage.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete network (should unlink addresses)
	if err := storage.DeleteNetwork(network.ID); err != nil {
		t.Fatalf("DeleteNetwork failed: %v", err)
	}

	// Verify device still exists but address has no network
	retrieved, err := storage.GetDevice(device.ID)
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

	err := storage.DeleteNetwork("non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteNetwork("")
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
		if err := storage.CreateNetwork(network); err != nil {
			t.Fatalf("CreateNetwork failed: %v", err)
		}
	}

	// List all networks
	result, err := storage.ListNetworks(nil)
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
	storage.CreateNetwork(&model.Network{Name: "Production-1", Subnet: "192.168.1.0/24"})
	storage.CreateNetwork(&model.Network{Name: "Production-2", Subnet: "192.168.2.0/24"})
	storage.CreateNetwork(&model.Network{Name: "Staging", Subnet: "10.0.0.0/16"})

	// Filter by name
	result, err := storage.ListNetworks(&model.NetworkFilter{Name: "Production"})
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
	storage.CreateDatacenter(dc1)
	storage.CreateDatacenter(dc2)

	// Create networks
	storage.CreateNetwork(&model.Network{Name: "Network1", Subnet: "192.168.1.0/24", DatacenterID: dc1.ID})
	storage.CreateNetwork(&model.Network{Name: "Network2", Subnet: "192.168.2.0/24", DatacenterID: dc1.ID})
	storage.CreateNetwork(&model.Network{Name: "Network3", Subnet: "10.0.0.0/16", DatacenterID: dc2.ID})

	// Filter by datacenter
	result, err := storage.ListNetworks(&model.NetworkFilter{DatacenterID: dc1.ID})
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
	storage.CreateNetwork(&model.Network{Name: "Network1", Subnet: "192.168.1.0/24", VLANID: 100})
	storage.CreateNetwork(&model.Network{Name: "Network2", Subnet: "192.168.2.0/24", VLANID: 100})
	storage.CreateNetwork(&model.Network{Name: "Network3", Subnet: "10.0.0.0/16", VLANID: 200})

	// Filter by VLAN
	result, err := storage.ListNetworks(&model.NetworkFilter{VLANID: 100})
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

	result, err := storage.ListNetworks(nil)
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
	if err := storage.CreateNetwork(network); err != nil {
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

	storage.CreateDevice(device1)
	storage.CreateDevice(device2)
	storage.CreateDevice(device3)

	// Get devices in network
	devices, err := storage.GetNetworkDevices(network.ID)
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

	_, err := storage.GetNetworkDevices("non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkDevicesEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with no devices
	network := &model.Network{Name: "Empty-Network", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	devices, err := storage.GetNetworkDevices(network.ID)
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

	_, err := storage.GetNetworkDevices("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkUtilization(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with /24 subnet (254 usable IPs)
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(network); err != nil {
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
	if err := storage.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Get utilization
	util, err := storage.GetNetworkUtilization(network.ID)
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

	_, err := storage.GetNetworkUtilization("non-existent-id")
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkUtilizationInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkUtilization("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestNetworkOperations_GetNetworkUtilizationEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with no devices
	network := &model.Network{Name: "Empty-Network", Subnet: "10.0.0.0/16"}
	if err := storage.CreateNetwork(network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	util, err := storage.GetNetworkUtilization(network.ID)
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
		{"192.168.1.0/24", 254, false},   // 256 - 2 (network + broadcast)
		{"10.0.0.0/16", 65534, false},    // 65536 - 2
		{"192.168.1.0/30", 2, false},     // 4 - 2
		{"192.168.1.0/31", 2, false},     // Point-to-point link
		{"192.168.1.1/32", 1, false},     // Single host
		{"invalid", 0, true},             // Invalid CIDR
		{"192.168.1.0", 0, true},         // Missing prefix
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

	err := storage.CreateNetwork(nil)
	if err == nil {
		t.Error("expected error for nil network")
	}
}

func TestNetworkOperations_UpdateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateNetwork(nil)
	if err == nil {
		t.Error("expected error for nil network")
	}
}

// ============================================================================
// Network Pool Operations Tests (P2-007)
// ============================================================================

func TestPoolOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network first
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(network); err != nil {
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
	err := storage.CreateNetworkPool(pool)
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
	retrieved, err := storage.GetNetworkPool(pool.ID)
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

	err := storage.CreateNetworkPool(pool)
	if err != ErrNetworkNotFound {
		t.Errorf("expected ErrNetworkNotFound, got %v", err)
	}
}

func TestPoolOperations_CreateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateNetworkPool(nil)
	if err == nil {
		t.Error("expected error for nil pool")
	}
}

func TestPoolOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkPool("non-existent-id")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNetworkPool("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Original Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.150",
		Tags:      []string{"original"},
	}
	if err := storage.CreateNetworkPool(pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	originalUpdatedAt := pool.UpdatedAt

	// Update pool
	pool.Name = "Updated Pool"
	pool.StartIP = "192.168.1.50"
	pool.EndIP = "192.168.1.200"
	pool.Description = "Updated description"
	pool.Tags = []string{"updated", "production"}

	if err := storage.UpdateNetworkPool(pool); err != nil {
		t.Fatalf("UpdateNetworkPool failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetNetworkPool(pool.ID)
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

	err := storage.UpdateNetworkPool(pool)
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_UpdateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateNetworkPool(nil)
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

	err := storage.UpdateNetworkPool(pool)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool to delete",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
		Tags:      []string{"delete-me"},
	}
	if err := storage.CreateNetworkPool(pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	// Delete pool
	if err := storage.DeleteNetworkPool(pool.ID); err != nil {
		t.Fatalf("DeleteNetworkPool failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetNetworkPool(pool.ID)
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(pool)

	// Create device with address in this pool
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.150", Type: "ipv4", PoolID: pool.ID},
		},
	}
	if err := storage.CreateDevice(device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete pool (should unlink addresses)
	if err := storage.DeleteNetworkPool(pool.ID); err != nil {
		t.Fatalf("DeleteNetworkPool failed: %v", err)
	}

	// Verify device still exists but address has no pool
	retrieved, err := storage.GetDevice(device.ID)
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

	err := storage.DeleteNetworkPool("non-existent-id")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteNetworkPool("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// Create multiple pools
	pools := []string{"Pool1", "Pool2", "Pool3"}
	for i, name := range pools {
		pool := &model.NetworkPool{
			NetworkID: network.ID,
			Name:      name,
			StartIP:   "192.168.1." + string(rune('1'+i)) + "00",
			EndIP:     "192.168.1." + string(rune('1'+i)) + "50",
		}
		if err := storage.CreateNetworkPool(pool); err != nil {
			t.Fatalf("CreateNetworkPool failed: %v", err)
		}
	}

	// List all pools
	result, err := storage.ListNetworkPools(nil)
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
	storage.CreateNetwork(network1)
	storage.CreateNetwork(network2)

	// Create pools in different networks
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network1.ID, Name: "Pool1", StartIP: "192.168.1.100", EndIP: "192.168.1.200",
	})
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network1.ID, Name: "Pool2", StartIP: "192.168.1.201", EndIP: "192.168.1.250",
	})
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network2.ID, Name: "Pool3", StartIP: "192.168.2.100", EndIP: "192.168.2.200",
	})

	// Filter by network
	result, err := storage.ListNetworkPools(&model.NetworkPoolFilter{NetworkID: network1.ID})
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
	storage.CreateNetwork(network)

	// Create pools with different tags
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network.ID, Name: "Pool1", StartIP: "192.168.1.100", EndIP: "192.168.1.150",
		Tags: []string{"dhcp", "production"},
	})
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network.ID, Name: "Pool2", StartIP: "192.168.1.151", EndIP: "192.168.1.200",
		Tags: []string{"dhcp", "staging"},
	})
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network.ID, Name: "Pool3", StartIP: "192.168.1.201", EndIP: "192.168.1.250",
		Tags: []string{"static"},
	})

	// Filter by single tag
	result, err := storage.ListNetworkPools(&model.NetworkPoolFilter{Tags: []string{"dhcp"}})
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 pools with 'dhcp' tag, got %d", len(result))
	}

	// Filter by multiple tags (AND logic)
	result, err = storage.ListNetworkPools(&model.NetworkPoolFilter{Tags: []string{"dhcp", "production"}})
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

	result, err := storage.ListNetworkPools(nil)
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Minimal Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	if err := storage.CreateNetworkPool(pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	// Get pool
	retrieved, err := storage.GetNetworkPool(pool.ID)
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "DHCP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.105",
	}
	storage.CreateNetworkPool(pool)

	// Get first available IP
	ip, err := storage.GetNextAvailableIP(pool.ID)
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
	storage.CreateDevice(device)

	// Get next available IP (should skip used one)
	ip, err = storage.GetNextAvailableIP(pool.ID)
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Small Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.101",
	}
	storage.CreateNetworkPool(pool)

	// Use all IPs
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.101", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(device)

	// Try to get next available IP
	_, err := storage.GetNextAvailableIP(pool.ID)
	if err != ErrIPNotAvailable {
		t.Errorf("expected ErrIPNotAvailable, got %v", err)
	}
}

func TestPoolOperations_GetNextAvailableIP_PoolNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNextAvailableIP("non-existent-pool")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_GetNextAvailableIP_InvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNextAvailableIP("")
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "DHCP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(pool)

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
			valid, err := storage.ValidateIPInPool(pool.ID, tt.ip)
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

	_, err := storage.ValidateIPInPool("non-existent-pool", "192.168.1.100")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_ValidateIPInPool_InvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.ValidateIPInPool("", "192.168.1.100")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_ValidateIPInPool_InvalidIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "DHCP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(pool)

	_, err := storage.ValidateIPInPool(pool.ID, "invalid-ip")
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Small Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.103",
	}
	storage.CreateNetworkPool(pool)

	// Create device with some addresses in the pool
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.102", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(device)

	// Get heatmap
	heatmap, err := storage.GetPoolHeatmap(pool.ID)
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Empty Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.102",
	}
	storage.CreateNetworkPool(pool)

	heatmap, err := storage.GetPoolHeatmap(pool.ID)
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

	_, err := storage.GetPoolHeatmap("non-existent-pool")
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestPoolOperations_GetPoolHeatmap_InvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetPoolHeatmap("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestPoolOperations_GetPoolHeatmap_SingleIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool with single IP
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Single IP Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.100",
	}
	storage.CreateNetworkPool(pool)

	heatmap, err := storage.GetPoolHeatmap(pool.ID)
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
		name        string
		ip          string
		endIP       string
		expectedIP  string
		expectedOK  bool
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

// Relationship tests

func TestRelationshipCRUD(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Create two devices
	device1 := &model.Device{Name: "Server1"}
	device2 := &model.Device{Name: "Server2"}
	if err := storage.CreateDevice(device1); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}
	if err := storage.CreateDevice(device2); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Add relationship
	if err := storage.AddRelationship(device1.ID, device2.ID, model.RelationshipContains); err != nil {
		t.Fatalf("AddRelationship failed: %v", err)
	}

	// Get relationships
	rels, err := storage.GetRelationships(device1.ID)
	if err != nil {
		t.Fatalf("GetRelationships failed: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rels))
	}
	if rels[0].ParentID != device1.ID || rels[0].ChildID != device2.ID {
		t.Errorf("relationship IDs mismatch")
	}

	// Remove relationship
	if err := storage.RemoveRelationship(device1.ID, device2.ID, model.RelationshipContains); err != nil {
		t.Fatalf("RemoveRelationship failed: %v", err)
	}

	// Verify removed
	rels, err = storage.GetRelationships(device1.ID)
	if err != nil {
		t.Fatalf("GetRelationships failed: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships after removal, got %d", len(rels))
	}
}

func TestGetRelatedDevices(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Create devices
	parent := &model.Device{Name: "Parent"}
	child1 := &model.Device{Name: "Child1"}
	child2 := &model.Device{Name: "Child2"}
	for _, d := range []*model.Device{parent, child1, child2} {
		if err := storage.CreateDevice(d); err != nil {
			t.Fatalf("CreateDevice failed: %v", err)
		}
	}

	// Add relationships
	storage.AddRelationship(parent.ID, child1.ID, model.RelationshipContains)
	storage.AddRelationship(parent.ID, child2.ID, model.RelationshipConnectedTo)

	// Get related by type
	related, err := storage.GetRelatedDevices(parent.ID, model.RelationshipContains)
	if err != nil {
		t.Fatalf("GetRelatedDevices failed: %v", err)
	}
	if len(related) != 1 || related[0].ID != child1.ID {
		t.Errorf("expected child1, got %v", related)
	}
}

func TestAddRelationshipIdempotent(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	device1 := &model.Device{Name: "D1"}
	device2 := &model.Device{Name: "D2"}
	storage.CreateDevice(device1)
	storage.CreateDevice(device2)

	// Add same relationship twice - should not error
	if err := storage.AddRelationship(device1.ID, device2.ID, model.RelationshipContains); err != nil {
		t.Fatalf("first AddRelationship failed: %v", err)
	}
	if err := storage.AddRelationship(device1.ID, device2.ID, model.RelationshipContains); err != nil {
		t.Fatalf("second AddRelationship failed: %v", err)
	}

	// Should still have only one relationship
	rels, _ := storage.GetRelationships(device1.ID)
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rels))
	}
}

// Discovery Storage Tests

func TestDiscoveredDeviceCRUD(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Create network first
	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(network); err != nil {
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
	if err := storage.CreateDiscoveredDevice(device); err != nil {
		t.Fatalf("CreateDiscoveredDevice failed: %v", err)
	}
	if device.ID == "" {
		t.Error("device ID should be set")
	}

	// Get device
	got, err := storage.GetDiscoveredDevice(device.ID)
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
	if err := storage.UpdateDiscoveredDevice(device); err != nil {
		t.Fatalf("UpdateDiscoveredDevice failed: %v", err)
	}
	got, _ = storage.GetDiscoveredDevice(device.ID)
	if got.Hostname != "updated-host" || got.Confidence != 95 {
		t.Errorf("update failed: got %+v", got)
	}

	// Delete device
	if err := storage.DeleteDiscoveredDevice(device.ID); err != nil {
		t.Fatalf("DeleteDiscoveredDevice failed: %v", err)
	}
	_, err = storage.GetDiscoveredDevice(device.ID)
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
	storage.CreateNetwork(network)

	device := &model.DiscoveredDevice{
		IP:        "192.168.1.50",
		NetworkID: network.ID,
		Status:    "active",
	}
	storage.CreateDiscoveredDevice(device)

	got, err := storage.GetDiscoveredDeviceByIP(network.ID, "192.168.1.50")
	if err != nil {
		t.Fatalf("GetDiscoveredDeviceByIP failed: %v", err)
	}
	if got.ID != device.ID {
		t.Errorf("device ID mismatch")
	}

	// Not found
	_, err = storage.GetDiscoveredDeviceByIP(network.ID, "192.168.1.99")
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
	storage.CreateNetwork(network1)
	storage.CreateNetwork(network2)

	storage.CreateDiscoveredDevice(&model.DiscoveredDevice{IP: "192.168.1.1", NetworkID: network1.ID})
	storage.CreateDiscoveredDevice(&model.DiscoveredDevice{IP: "192.168.1.2", NetworkID: network1.ID})
	storage.CreateDiscoveredDevice(&model.DiscoveredDevice{IP: "192.168.2.1", NetworkID: network2.ID})

	// List all
	all, err := storage.ListDiscoveredDevices("")
	if err != nil {
		t.Fatalf("ListDiscoveredDevices failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 devices, got %d", len(all))
	}

	// List by network
	net1Devices, err := storage.ListDiscoveredDevices(network1.ID)
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
	storage.CreateNetwork(network)

	discovered := &model.DiscoveredDevice{IP: "192.168.1.10", NetworkID: network.ID}
	storage.CreateDiscoveredDevice(discovered)

	device := &model.Device{Name: "Promoted Device"}
	storage.CreateDevice(device)

	if err := storage.PromoteDiscoveredDevice(discovered.ID, device.ID); err != nil {
		t.Fatalf("PromoteDiscoveredDevice failed: %v", err)
	}

	got, _ := storage.GetDiscoveredDevice(discovered.ID)
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
	storage.CreateNetwork(network)

	scan := &model.DiscoveryScan{
		NetworkID:  network.ID,
		Status:     model.ScanStatusPending,
		ScanType:   model.ScanTypeFull,
		TotalHosts: 254,
	}
	if err := storage.CreateDiscoveryScan(scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}
	if scan.ID == "" {
		t.Error("scan ID should be set")
	}

	got, err := storage.GetDiscoveryScan(scan.ID)
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
	if err := storage.UpdateDiscoveryScan(scan); err != nil {
		t.Fatalf("UpdateDiscoveryScan failed: %v", err)
	}
	got, _ = storage.GetDiscoveryScan(scan.ID)
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
	storage.CreateNetwork(network)

	storage.CreateDiscoveryScan(&model.DiscoveryScan{NetworkID: network.ID, Status: model.ScanStatusCompleted})
	storage.CreateDiscoveryScan(&model.DiscoveryScan{NetworkID: network.ID, Status: model.ScanStatusRunning})

	scans, err := storage.ListDiscoveryScans(network.ID)
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
	storage.CreateNetwork(network)

	rule := &model.DiscoveryRule{
		NetworkID:     network.ID,
		Enabled:       true,
		ScanType:      model.ScanTypeFull,
		IntervalHours: 24,
		ExcludeIPs:    "192.168.1.1,192.168.1.254",
	}
	if err := storage.SaveDiscoveryRule(rule); err != nil {
		t.Fatalf("SaveDiscoveryRule failed: %v", err)
	}

	got, err := storage.GetDiscoveryRule(network.ID)
	if err != nil {
		t.Fatalf("GetDiscoveryRule failed: %v", err)
	}
	if !got.Enabled || got.IntervalHours != 24 {
		t.Errorf("rule mismatch: got %+v", got)
	}

	// Update rule (upsert)
	rule.Enabled = false
	rule.IntervalHours = 12
	if err := storage.SaveDiscoveryRule(rule); err != nil {
		t.Fatalf("SaveDiscoveryRule update failed: %v", err)
	}
	got, _ = storage.GetDiscoveryRule(network.ID)
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
	storage.CreateNetwork(network1)
	storage.CreateNetwork(network2)

	storage.SaveDiscoveryRule(&model.DiscoveryRule{NetworkID: network1.ID, Enabled: true})
	storage.SaveDiscoveryRule(&model.DiscoveryRule{NetworkID: network2.ID, Enabled: false})

	rules, err := storage.ListDiscoveryRules()
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
	storage.CreateNetwork(network)

	// Create devices - one will be promoted
	storage.CreateDiscoveredDevice(&model.DiscoveredDevice{IP: "192.168.1.1", NetworkID: network.ID})
	promoted := &model.DiscoveredDevice{IP: "192.168.1.2", NetworkID: network.ID}
	storage.CreateDiscoveredDevice(promoted)

	device := &model.Device{Name: "Promoted"}
	storage.CreateDevice(device)
	storage.PromoteDiscoveredDevice(promoted.ID, device.ID)

	// Cleanup with 0 days should remove non-promoted devices
	if err := storage.CleanupOldDiscoveries(0); err != nil {
		t.Fatalf("CleanupOldDiscoveries failed: %v", err)
	}

	devices, _ := storage.ListDiscoveredDevices(network.ID)
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

	_, err = storage.GetDiscoveredDevice("nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	_, err = storage.GetDiscoveryScan("nonexistent")
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}

	_, err = storage.GetDiscoveryRule("nonexistent")
	if err != ErrRuleNotFound {
		t.Errorf("expected ErrRuleNotFound, got %v", err)
	}

	err = storage.UpdateDiscoveredDevice(&model.DiscoveredDevice{ID: "nonexistent"})
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	err = storage.DeleteDiscoveredDevice("nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	err = storage.PromoteDiscoveredDevice("nonexistent", "device-id")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	err = storage.UpdateDiscoveryScan(&model.DiscoveryScan{ID: "nonexistent"})
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}
}

// ============================================================================
// Additional Tests for Coverage (P2-011)
// ============================================================================

func TestDBMethod(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	db := storage.DB()
	if db == nil {
		t.Error("DB() should return non-nil database")
	}

	// Verify it's the same connection
	if err := db.Ping(); err != nil {
		t.Errorf("DB() returned invalid connection: %v", err)
	}
}

func TestCreateDeviceNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateDevice(nil)
	if err == nil {
		t.Error("expected error for nil device")
	}
}

func TestUpdateDeviceNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateDevice(nil)
	if err == nil {
		t.Error("expected error for nil device")
	}
}

func TestUpdateDeviceInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{ID: "", Name: "test"}
	err := storage.UpdateDevice(device)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestCreateDatacenterNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateDatacenter(nil)
	if err == nil {
		t.Error("expected error for nil datacenter")
	}
}

func TestUpdateDatacenterNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateDatacenter(nil)
	if err == nil {
		t.Error("expected error for nil datacenter")
	}
}

func TestUpdateDatacenterInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	dc := &model.Datacenter{ID: "", Name: "test"}
	err := storage.UpdateDatacenter(dc)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestRelationshipInvalidIDs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create valid devices for testing
	device1 := &model.Device{Name: "D1"}
	device2 := &model.Device{Name: "D2"}
	storage.CreateDevice(device1)
	storage.CreateDevice(device2)

	// Test with non-existent device IDs (FK constraint)
	err := storage.AddRelationship("nonexistent1", "nonexistent2", model.RelationshipContains)
	if err == nil {
		t.Error("expected error for non-existent device IDs")
	}

	// Valid relationship should work
	err = storage.AddRelationship(device1.ID, device2.ID, model.RelationshipContains)
	if err != nil {
		t.Errorf("AddRelationship failed: %v", err)
	}

	// GetRelationships with valid ID
	rels, err := storage.GetRelationships(device1.ID)
	if err != nil {
		t.Errorf("GetRelationships failed: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rels))
	}

	// GetRelatedDevices with valid ID
	related, err := storage.GetRelatedDevices(device1.ID, model.RelationshipContains)
	if err != nil {
		t.Errorf("GetRelatedDevices failed: %v", err)
	}
	if len(related) != 1 {
		t.Errorf("expected 1 related device, got %d", len(related))
	}

	// RemoveRelationship
	err = storage.RemoveRelationship(device1.ID, device2.ID, model.RelationshipContains)
	if err != nil {
		t.Errorf("RemoveRelationship failed: %v", err)
	}
}

func TestDiscoveryInvalidIDs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// UpdateDiscoveredDevice with non-existent ID
	err := storage.UpdateDiscoveredDevice(&model.DiscoveredDevice{ID: "nonexistent", IP: "192.168.1.1", NetworkID: network.ID})
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// GetDiscoveredDevice with non-existent ID returns not found
	_, err = storage.GetDiscoveredDevice("nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// DeleteDiscoveredDevice with non-existent ID returns not found
	err = storage.DeleteDiscoveredDevice("nonexistent")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// PromoteDiscoveredDevice with non-existent discovered ID
	err = storage.PromoteDiscoveredDevice("nonexistent", "device")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}

	// GetDiscoveredDeviceByIP with non-existent network returns not found
	_, err = storage.GetDiscoveredDeviceByIP("nonexistent", "192.168.1.1")
	if err != ErrDiscoveryNotFound {
		t.Errorf("expected ErrDiscoveryNotFound, got %v", err)
	}
}

func TestDiscoveryScanInvalidIDs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// UpdateDiscoveryScan with non-existent ID
	err := storage.UpdateDiscoveryScan(&model.DiscoveryScan{ID: "nonexistent", NetworkID: network.ID})
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}

	// GetDiscoveryScan with non-existent ID
	_, err = storage.GetDiscoveryScan("nonexistent")
	if err != ErrScanNotFound {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}
}

func TestDiscoveryRuleInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// GetDiscoveryRule with non-existent network ID
	_, err := storage.GetDiscoveryRule("nonexistent")
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
	err = storage.SaveDiscoveryRule(rule)
	if err != nil {
		t.Errorf("SaveDiscoveryRule failed: %v", err)
	}

	// Verify rule was saved
	got, err := storage.GetDiscoveryRule(network.ID)
	if err != nil {
		t.Errorf("GetDiscoveryRule failed: %v", err)
	}
	if !got.Enabled {
		t.Error("expected rule to be enabled")
	}
}

func TestListDevicesWithNetworkFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network
	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// Create devices with addresses in network
	device1 := &model.Device{
		Name:      "server1",
		Addresses: []model.Address{{IP: "192.168.1.100", Type: "ipv4", NetworkID: network.ID}},
	}
	device2 := &model.Device{
		Name:      "server2",
		Addresses: []model.Address{{IP: "10.0.0.1", Type: "ipv4"}},
	}
	storage.CreateDevice(device1)
	storage.CreateDevice(device2)

	// Filter by network
	result, err := storage.ListDevices(&model.DeviceFilter{NetworkID: network.ID})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 device in network, got %d", len(result))
	}
}

func TestNetworkUtilizationInvalidSubnet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with invalid subnet
	network := &model.Network{Name: "BadNet", Subnet: "invalid-cidr"}
	storage.CreateNetwork(network)

	_, err := storage.GetNetworkUtilization(network.ID)
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestPoolOperations_LargeRange(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool with larger range
	network := &model.Network{Name: "Network1", Subnet: "10.0.0.0/16"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Large Pool",
		StartIP:   "10.0.1.0",
		EndIP:     "10.0.1.255",
	}
	storage.CreateNetworkPool(pool)

	// Get first available IP
	ip, err := storage.GetNextAvailableIP(pool.ID)
	if err != nil {
		t.Fatalf("GetNextAvailableIP failed: %v", err)
	}
	if ip != "10.0.1.0" {
		t.Errorf("expected '10.0.1.0', got '%s'", ip)
	}
}

func TestDeleteNetworkWithPools(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with pool
	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(pool)

	// Delete network (should cascade to pools)
	if err := storage.DeleteNetwork(network.ID); err != nil {
		t.Fatalf("DeleteNetwork failed: %v", err)
	}

	// Verify pool is deleted
	_, err := storage.GetNetworkPool(pool.ID)
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound after network deletion, got %v", err)
	}
}

func TestDeviceWithAllFields(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter and network
	dc := &model.Datacenter{Name: "DC1", Location: "NYC"}
	storage.CreateDatacenter(dc)

	network := &model.Network{Name: "Net1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(pool)

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
			{IP: "192.168.1.100", Port: 22, Type: "ipv4", Label: "primary", NetworkID: network.ID, PoolID: pool.ID},
			{IP: "192.168.1.101", Port: 443, Type: "ipv4", Label: "secondary", NetworkID: network.ID},
			{IP: "2001:db8::1", Type: "ipv6", Label: "ipv6"},
		},
		Domains: []string{"server.example.com", "www.example.com", "api.example.com"},
	}

	if err := storage.CreateDevice(device); err != nil {
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
	storage.CreateDevice(device)

	// Search should handle special SQL characters
	result, err := storage.SearchDevices("server-01")
	if err != nil {
		t.Fatalf("SearchDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestListDatacentersWithNameFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenters with different names
	storage.CreateDatacenter(&model.Datacenter{Name: "NYC-DC1", Location: "New York"})
	storage.CreateDatacenter(&model.Datacenter{Name: "NYC-DC2", Location: "New York"})
	storage.CreateDatacenter(&model.Datacenter{Name: "LA-DC1", Location: "Los Angeles"})

	// Filter by name prefix
	result, err := storage.ListDatacenters(&model.DatacenterFilter{Name: "NYC"})
	if err != nil {
		t.Fatalf("ListDatacenters failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 datacenters matching NYC, got %d", len(result))
	}
}

func TestCleanupOldDiscoveriesWithDays(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// Create discovered device
	storage.CreateDiscoveredDevice(&model.DiscoveredDevice{IP: "192.168.1.1", NetworkID: network.ID})

	// Cleanup with 30 days should not remove recent devices
	if err := storage.CleanupOldDiscoveries(30); err != nil {
		t.Fatalf("CleanupOldDiscoveries failed: %v", err)
	}

	devices, _ := storage.ListDiscoveredDevices(network.ID)
	if len(devices) != 1 {
		t.Errorf("expected 1 device (recent), got %d", len(devices))
	}
}

func TestRemoveRelationshipNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices but no relationship
	device1 := &model.Device{Name: "D1"}
	device2 := &model.Device{Name: "D2"}
	storage.CreateDevice(device1)
	storage.CreateDevice(device2)

	// Remove non-existent relationship should not error (idempotent)
	err := storage.RemoveRelationship(device1.ID, device2.ID, model.RelationshipContains)
	if err != nil {
		t.Errorf("RemoveRelationship should be idempotent, got %v", err)
	}
}

func TestGetRelatedDevicesEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{Name: "Lonely"}
	storage.CreateDevice(device)

	related, err := storage.GetRelatedDevices(device.ID, model.RelationshipContains)
	if err != nil {
		t.Fatalf("GetRelatedDevices failed: %v", err)
	}
	if len(related) != 0 {
		t.Errorf("expected 0 related devices, got %d", len(related))
	}
}

func TestDiscoveryScanWithAllFields(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

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
	if err := storage.CreateDiscoveryScan(scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}

	// Complete the scan
	scan.Status = model.ScanStatusCompleted
	scan.ScannedHosts = 254
	scan.FoundHosts = 50
	scan.ProgressPercent = 100.0
	if err := storage.UpdateDiscoveryScan(scan); err != nil {
		t.Fatalf("UpdateDiscoveryScan failed: %v", err)
	}

	got, _ := storage.GetDiscoveryScan(scan.ID)
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
	storage.CreateNetwork(network)

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
	storage.CreateDiscoveredDevice(device)

	got, _ := storage.GetDiscoveredDevice(device.ID)
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
	storage.CreateNetwork(network)

	scans, err := storage.ListDiscoveryScans(network.ID)
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

	rules, err := storage.ListDiscoveryRules()
	if err != nil {
		t.Fatalf("ListDiscoveryRules failed: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestDeleteDeviceWithRelationships(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with relationships
	parent := &model.Device{Name: "Parent"}
	child := &model.Device{Name: "Child"}
	storage.CreateDevice(parent)
	storage.CreateDevice(child)
	storage.AddRelationship(parent.ID, child.ID, model.RelationshipContains)

	// Delete parent - should cascade relationships
	if err := storage.DeleteDevice(parent.ID); err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	// Verify relationship is gone
	rels, _ := storage.GetRelationships(child.ID)
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships after parent deletion, got %d", len(rels))
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestDiscoveryScanWithError(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	// Create scan with error message
	scan := &model.DiscoveryScan{
		NetworkID:    network.ID,
		Status:       model.ScanStatusFailed,
		ScanType:     model.ScanTypeFull,
		TotalHosts:   254,
		ErrorMessage: "Connection timeout",
	}
	if err := storage.CreateDiscoveryScan(scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}

	got, _ := storage.GetDiscoveryScan(scan.ID)
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
	storage.CreateNetwork(network1)
	storage.CreateNetwork(network2)

	storage.CreateDiscoveryScan(&model.DiscoveryScan{NetworkID: network1.ID, Status: model.ScanStatusCompleted})
	storage.CreateDiscoveryScan(&model.DiscoveryScan{NetworkID: network2.ID, Status: model.ScanStatusCompleted})

	// List all scans (empty network ID)
	scans, err := storage.ListDiscoveryScans("")
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
	storage.CreateNetwork(network)

	device := &model.DiscoveredDevice{
		IP:         "192.168.1.50",
		NetworkID:  network.ID,
		Status:     "active",
		Confidence: 50,
	}
	storage.CreateDiscoveredDevice(device)

	// Update with new data
	device.Hostname = "updated-host"
	device.Confidence = 95
	device.OSGuess = "Linux"
	device.Vendor = "Dell"
	device.OpenPorts = []int{22, 80}
	device.Services = []model.ServiceInfo{{Port: 22, Service: "ssh"}}

	if err := storage.UpdateDiscoveredDevice(device); err != nil {
		t.Fatalf("UpdateDiscoveredDevice failed: %v", err)
	}

	got, _ := storage.GetDiscoveredDevice(device.ID)
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

func TestNetworkPoolUpdateTags(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
		Tags:      []string{"original"},
	}
	storage.CreateNetworkPool(pool)

	// Update with new tags
	pool.Tags = []string{"updated", "new-tag", "another"}
	if err := storage.UpdateNetworkPool(pool); err != nil {
		t.Fatalf("UpdateNetworkPool failed: %v", err)
	}

	got, _ := storage.GetNetworkPool(pool.ID)
	if len(got.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(got.Tags))
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
	storage.CreateDevice(device)

	// Update to clear arrays
	device.Tags = []string{}
	device.Domains = []string{}
	device.Addresses = []model.Address{}
	if err := storage.UpdateDevice(device); err != nil {
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

func TestPoolUpdateClearTags(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
		Tags:      []string{"tag1", "tag2"},
	}
	storage.CreateNetworkPool(pool)

	// Update to clear tags
	pool.Tags = []string{}
	if err := storage.UpdateNetworkPool(pool); err != nil {
		t.Fatalf("UpdateNetworkPool failed: %v", err)
	}

	got, _ := storage.GetNetworkPool(pool.ID)
	if len(got.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(got.Tags))
	}
}

func TestCalculateCIDRSizeEdgeCases(t *testing.T) {
	tests := []struct {
		cidr     string
		expected int
		hasError bool
	}{
		{"10.0.0.0/8", 1 << 20, false},    // Large network (capped at ~1M)
		{"192.168.1.128/25", 126, false},  // /25 subnet
		{"192.168.1.192/26", 62, false},   // /26 subnet
		{"192.168.1.240/28", 14, false},   // /28 subnet
		{"192.168.1.252/30", 2, false},    // /30 subnet
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

func TestGetNextAvailableIPWithGaps(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "Network1", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.110",
	}
	storage.CreateNetworkPool(pool)

	// Use IPs with gaps: 100, 102, 104
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.102", Type: "ipv4", PoolID: pool.ID},
			{IP: "192.168.1.104", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(device)

	// Should return first gap: 101
	ip, err := storage.GetNextAvailableIP(pool.ID)
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
	storage.CreateNetwork(network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Pool1",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.102",
	}
	storage.CreateNetworkPool(pool)

	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.101", Type: "ipv4", PoolID: pool.ID},
		},
	}
	storage.CreateDevice(device)

	heatmap, err := storage.GetPoolHeatmap(pool.ID)
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
	storage.CreateNetwork(network)

	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network.ID, Name: "DHCP-Pool", StartIP: "192.168.1.100", EndIP: "192.168.1.150",
		Tags: []string{"dhcp", "production"},
	})
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network.ID, Name: "Static-Pool", StartIP: "192.168.1.151", EndIP: "192.168.1.200",
		Tags: []string{"static", "production"},
	})
	storage.CreateNetworkPool(&model.NetworkPool{
		NetworkID: network.ID, Name: "DHCP-Reserved", StartIP: "192.168.1.201", EndIP: "192.168.1.250",
		Tags: []string{"dhcp", "reserved"},
	})

	// Filter by network and tags
	result, err := storage.ListNetworkPools(&model.NetworkPoolFilter{NetworkID: network.ID, Tags: []string{"dhcp"}})
	if err != nil {
		t.Fatalf("ListNetworkPools failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 pools with 'dhcp' tag, got %d", len(result))
	}
}

func TestDiscoveryRuleWithAllFields(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(network)

	rule := &model.DiscoveryRule{
		NetworkID:     network.ID,
		Enabled:       true,
		ScanType:      model.ScanTypeDeep,
		IntervalHours: 12,
		ExcludeIPs:    "192.168.1.1,192.168.1.254",
	}
	storage.SaveDiscoveryRule(rule)

	got, _ := storage.GetDiscoveryRule(network.ID)
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
	storage.CreateNetwork(network1)
	storage.CreateNetwork(network2)
	storage.CreateNetwork(network3)

	storage.SaveDiscoveryRule(&model.DiscoveryRule{NetworkID: network1.ID, Enabled: true, ScanType: model.ScanTypeQuick})
	storage.SaveDiscoveryRule(&model.DiscoveryRule{NetworkID: network2.ID, Enabled: false, ScanType: model.ScanTypeFull})
	storage.SaveDiscoveryRule(&model.DiscoveryRule{NetworkID: network3.ID, Enabled: true, ScanType: model.ScanTypeDeep})

	rules, err := storage.ListDiscoveryRules()
	if err != nil {
		t.Fatalf("ListDiscoveryRules failed: %v", err)
	}
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestSearchDevicesMultipleMatches(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with overlapping search terms
	storage.CreateDevice(&model.Device{
		Name:        "web-server-1",
		Description: "Production web server",
		Tags:        []string{"production", "web"},
	})
	storage.CreateDevice(&model.Device{
		Name:        "web-server-2",
		Description: "Staging web server",
		Tags:        []string{"staging", "web"},
	})
	storage.CreateDevice(&model.Device{
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

func TestNetworkWithZeroVLAN(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network with VLAN 0 (untagged)
	network := &model.Network{
		Name:   "Untagged Network",
		Subnet: "192.168.1.0/24",
		VLANID: 0,
	}
	storage.CreateNetwork(network)

	got, _ := storage.GetNetwork(network.ID)
	if got.VLANID != 0 {
		t.Errorf("expected VLAN 0, got %d", got.VLANID)
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
	storage.CreateDevice(device)

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
