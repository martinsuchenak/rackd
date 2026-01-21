package storage

import (
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
