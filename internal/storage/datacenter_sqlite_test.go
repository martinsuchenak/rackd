package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Datacenter Operations Tests
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
	err := storage.CreateDatacenter(context.Background(), dc)
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
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	originalUpdatedAt := dc.UpdatedAt

	// Update datacenter
	dc.Name = "DC1-Updated"
	dc.Location = "Chicago"
	dc.Description = "Updated description"

	if err := storage.UpdateDatacenter(context.Background(), dc); err != nil {
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

	err := storage.UpdateDatacenter(context.Background(), dc)
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound, got %v", err)
	}
}

func TestDatacenterOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "DC-to-delete"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Delete datacenter
	if err := storage.DeleteDatacenter(context.Background(), dc.ID); err != nil {
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
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create devices in datacenter
	device := &model.Device{Name: "server1", DatacenterID: dc.ID}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Delete datacenter (should unlink devices)
	if err := storage.DeleteDatacenter(context.Background(), dc.ID); err != nil {
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

	err := storage.DeleteDatacenter(context.Background(), "non-existent-id")
	if err != ErrDatacenterNotFound {
		t.Errorf("expected ErrDatacenterNotFound, got %v", err)
	}
}

func TestDatacenterOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteDatacenter(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestDatacenterOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Remove default datacenter to start clean
	defaultDCs, _ := storage.ListDatacenters(&model.DatacenterFilter{Name: "Default"})
	for _, dc := range defaultDCs {
		storage.DeleteDatacenter(context.Background(), dc.ID)
	}

	// Create multiple datacenters
	names := []string{"DC1", "DC2", "DC3"}
	for _, name := range names {
		dc := &model.Datacenter{Name: name}
		if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
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
	storage.CreateDatacenter(context.Background(), &model.Datacenter{Name: "NYC-DC1"})
	storage.CreateDatacenter(context.Background(), &model.Datacenter{Name: "NYC-DC2"})
	storage.CreateDatacenter(context.Background(), &model.Datacenter{Name: "LA-DC1"})

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

	// Remove default datacenter to start clean
	defaultDCs, _ := storage.ListDatacenters(&model.DatacenterFilter{Name: "Default"})
	for _, dc := range defaultDCs {
		storage.DeleteDatacenter(context.Background(), dc.ID)
	}

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
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create devices in datacenter
	device1 := &model.Device{Name: "server1", DatacenterID: dc.ID}
	device2 := &model.Device{Name: "server2", DatacenterID: dc.ID}
	device3 := &model.Device{Name: "server3"} // Not in datacenter

	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)
	storage.CreateDevice(context.Background(), device3)

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
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
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

func TestCreateDatacenterNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateDatacenter(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil datacenter")
	}
}

func TestUpdateDatacenterNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateDatacenter(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil datacenter")
	}
}

func TestUpdateDatacenterInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	dc := &model.Datacenter{ID: "", Name: "test"}
	err := storage.UpdateDatacenter(context.Background(), dc)
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestListDatacentersWithNameFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenters with different names
	storage.CreateDatacenter(context.Background(), &model.Datacenter{Name: "NYC-DC1", Location: "New York"})
	storage.CreateDatacenter(context.Background(), &model.Datacenter{Name: "NYC-DC2", Location: "New York"})
	storage.CreateDatacenter(context.Background(), &model.Datacenter{Name: "LA-DC1", Location: "Los Angeles"})

	// Filter by name prefix
	result, err := storage.ListDatacenters(&model.DatacenterFilter{Name: "NYC"})
	if err != nil {
		t.Fatalf("ListDatacenters failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 datacenters matching NYC, got %d", len(result))
	}
}
