package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Conflict Operations Tests
// ============================================================================

func TestConflictOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	conflict := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "IP 192.168.1.100 assigned to multiple devices",
		IPAddress:   "192.168.1.100",
		DeviceIDs:   []string{"device-1", "device-2"},
		DeviceNames: []string{"server1", "server2"},
	}

	// Create conflict
	err := storage.CreateConflict(context.Background(), conflict)
	if err != nil {
		t.Fatalf("CreateConflict failed: %v", err)
	}

	if conflict.ID == "" {
		t.Error("conflict ID should be set after creation")
	}
	if conflict.DetectedAt.IsZero() {
		t.Error("detected_at should be set after creation")
	}

	// Get conflict
	retrieved, err := storage.GetConflict(conflict.ID)
	if err != nil {
		t.Fatalf("GetConflict failed: %v", err)
	}

	if retrieved.Type != conflict.Type {
		t.Errorf("expected type %s, got %s", conflict.Type, retrieved.Type)
	}
	if retrieved.Status != conflict.Status {
		t.Errorf("expected status %s, got %s", conflict.Status, retrieved.Status)
	}
	if retrieved.Description != conflict.Description {
		t.Errorf("expected description %s, got %s", conflict.Description, retrieved.Description)
	}
	if retrieved.IPAddress != conflict.IPAddress {
		t.Errorf("expected ip_address %s, got %s", conflict.IPAddress, retrieved.IPAddress)
	}
	if len(retrieved.DeviceIDs) != 2 {
		t.Errorf("expected 2 device IDs, got %d", len(retrieved.DeviceIDs))
	}
	if len(retrieved.DeviceNames) != 2 {
		t.Errorf("expected 2 device names, got %d", len(retrieved.DeviceNames))
	}
}

func TestConflictOperations_CreateOverlappingSubnet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	conflict := &model.Conflict{
		Type:        model.ConflictTypeOverlappingSubnet,
		Status:      model.ConflictStatusActive,
		Description: "Subnets 10.0.0.0/24 and 10.0.0.0/16 overlap",
		NetworkIDs:   []string{"network-1", "network-2"},
		NetworkNames: []string{"Prod Network", "Dev Network"},
		Subnets:      []string{"10.0.0.0/24", "10.0.0.0/16"},
	}

	err := storage.CreateConflict(context.Background(), conflict)
	if err != nil {
		t.Fatalf("CreateConflict failed: %v", err)
	}

	retrieved, err := storage.GetConflict(conflict.ID)
	if err != nil {
		t.Fatalf("GetConflict failed: %v", err)
	}

	if retrieved.Type != model.ConflictTypeOverlappingSubnet {
		t.Errorf("expected type overlapping_subnet, got %s", retrieved.Type)
	}
	if len(retrieved.NetworkIDs) != 2 {
		t.Errorf("expected 2 network IDs, got %d", len(retrieved.NetworkIDs))
	}
	if len(retrieved.Subnets) != 2 {
		t.Errorf("expected 2 subnets, got %d", len(retrieved.Subnets))
	}
}

func TestConflictOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetConflict("non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent conflict")
	}
}

func TestConflictOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetConflict("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestConflictOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple conflicts
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Conflict 1",
		IPAddress:   "10.0.0.1",
	})
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeOverlappingSubnet,
		Status:      model.ConflictStatusResolved,
		Description: "Conflict 2",
	})

	// List all conflicts
	conflicts, err := storage.ListConflicts(nil)
	if err != nil {
		t.Fatalf("ListConflicts failed: %v", err)
	}

	if len(conflicts) != 2 {
		t.Errorf("expected 2 conflicts, got %d", len(conflicts))
	}
}

func TestConflictOperations_ListWithTypeFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create conflicts of different types
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Duplicate IP",
		IPAddress:   "10.0.0.1",
	})
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeOverlappingSubnet,
		Status:      model.ConflictStatusActive,
		Description: "Overlapping subnet",
	})

	// Filter by type
	conflicts, err := storage.ListConflicts(&model.ConflictFilter{
		Type: model.ConflictTypeDuplicateIP,
	})
	if err != nil {
		t.Fatalf("ListConflicts failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict matching type duplicate_ip, got %d", len(conflicts))
	}
	if conflicts[0].Type != model.ConflictTypeDuplicateIP {
		t.Errorf("expected type duplicate_ip, got %s", conflicts[0].Type)
	}
}

func TestConflictOperations_ListWithStatusFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create conflicts with different statuses
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Active conflict",
		IPAddress:   "10.0.0.1",
	})
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusResolved,
		Description: "Resolved conflict",
		IPAddress:   "10.0.0.2",
	})

	// Filter by status
	conflicts, err := storage.ListConflicts(&model.ConflictFilter{
		Status: model.ConflictStatusActive,
	})
	if err != nil {
		t.Fatalf("ListConflicts failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 active conflict, got %d", len(conflicts))
	}
	if conflicts[0].Status != model.ConflictStatusActive {
		t.Errorf("expected status active, got %s", conflicts[0].Status)
	}
}

func TestConflictOperations_ListEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	conflicts, err := storage.ListConflicts(nil)
	if err != nil {
		t.Fatalf("ListConflicts failed: %v", err)
	}

	if conflicts == nil {
		t.Error("conflicts should be empty slice, not nil")
	}
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestConflictOperations_UpdateStatus(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create conflict
	conflict := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Test conflict",
		IPAddress:   "10.0.0.1",
	}
	if err := storage.CreateConflict(context.Background(), conflict); err != nil {
		t.Fatalf("CreateConflict failed: %v", err)
	}

	// Update status to resolved
	err := storage.UpdateConflictStatus(context.Background(), conflict.ID, model.ConflictStatusResolved, "admin", "Fixed by removing duplicate IP")
	if err != nil {
		t.Fatalf("UpdateConflictStatus failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetConflict(conflict.ID)
	if err != nil {
		t.Fatalf("GetConflict failed: %v", err)
	}

	if retrieved.Status != model.ConflictStatusResolved {
		t.Errorf("expected status resolved, got %s", retrieved.Status)
	}
	if retrieved.ResolvedBy != "admin" {
		t.Errorf("expected resolved_by 'admin', got %s", retrieved.ResolvedBy)
	}
	if retrieved.Notes != "Fixed by removing duplicate IP" {
		t.Errorf("expected notes 'Fixed by removing duplicate IP', got %s", retrieved.Notes)
	}
	if retrieved.ResolvedAt == nil {
		t.Error("resolved_at should be set")
	}
}

func TestConflictOperations_UpdateStatusInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.UpdateConflictStatus(context.Background(), "", model.ConflictStatusResolved, "admin", "notes")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestConflictOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create conflict
	conflict := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Test conflict",
		IPAddress:   "10.0.0.1",
	}
	if err := storage.CreateConflict(context.Background(), conflict); err != nil {
		t.Fatalf("CreateConflict failed: %v", err)
	}

	// Delete conflict
	if err := storage.DeleteConflict(context.Background(), conflict.ID); err != nil {
		t.Fatalf("DeleteConflict failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetConflict(conflict.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestConflictOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteConflict(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestConflictOperations_FindDuplicateIPs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with duplicate IPs
	device1 := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4"},
		},
	}
	device2 := &model.Device{
		Name: "server2",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4"}, // Duplicate IP
		},
	}
	device3 := &model.Device{
		Name: "server3",
		Addresses: []model.Address{
			{IP: "192.168.1.101", Type: "ipv4"}, // Unique IP
		},
	}

	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)
	storage.CreateDevice(context.Background(), device3)

	// Find duplicate IPs
	conflicts, err := storage.FindDuplicateIPs(context.Background())
	if err != nil {
		t.Fatalf("FindDuplicateIPs failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}

	if len(conflicts) > 0 {
		conflict := conflicts[0]
		if conflict.Type != model.ConflictTypeDuplicateIP {
			t.Errorf("expected type duplicate_ip, got %s", conflict.Type)
		}
		if conflict.IPAddress != "192.168.1.100" {
			t.Errorf("expected IP 192.168.1.100, got %s", conflict.IPAddress)
		}
		if len(conflict.DeviceIDs) != 2 {
			t.Errorf("expected 2 device IDs, got %d", len(conflict.DeviceIDs))
		}
	}
}

func TestConflictOperations_FindOverlappingSubnets(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create networks with overlapping subnets
	network1 := &model.Network{Name: "Network1", Subnet: "10.0.0.0/24"}
	network2 := &model.Network{Name: "Network2", Subnet: "10.0.0.0/16"} // Overlaps with network1
	network3 := &model.Network{Name: "Network3", Subnet: "192.168.1.0/24"} // No overlap

	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)
	storage.CreateNetwork(context.Background(), network3)

	// Find overlapping subnets
	conflicts, err := storage.FindOverlappingSubnets(context.Background())
	if err != nil {
		t.Fatalf("FindOverlappingSubnets failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}

	if len(conflicts) > 0 {
		conflict := conflicts[0]
		if conflict.Type != model.ConflictTypeOverlappingSubnet {
			t.Errorf("expected type overlapping_subnet, got %s", conflict.Type)
		}
		if len(conflict.NetworkIDs) != 2 {
			t.Errorf("expected 2 network IDs, got %d", len(conflict.NetworkIDs))
		}
		if len(conflict.Subnets) != 2 {
			t.Errorf("expected 2 subnets, got %d", len(conflict.Subnets))
		}
	}
}

func TestConflictOperations_FindOverlappingSubnetsIdentical(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create networks with identical subnets (should still detect overlap)
	network1 := &model.Network{Name: "Network1", Subnet: "10.0.0.0/24"}
	network2 := &model.Network{Name: "Network2", Subnet: "10.0.0.0/24"}

	storage.CreateNetwork(context.Background(), network1)
	storage.CreateNetwork(context.Background(), network2)

	// Find overlapping subnets
	conflicts, err := storage.FindOverlappingSubnets(context.Background())
	if err != nil {
		t.Fatalf("FindOverlappingSubnets failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict for identical subnets, got %d", len(conflicts))
	}
}

func TestConflictOperations_FindOverlappingSubnetsNoOverlap(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create networks with non-overlapping subnets
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network1", Subnet: "10.0.0.0/24"})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network2", Subnet: "192.168.0.0/24"})
	storage.CreateNetwork(context.Background(), &model.Network{Name: "Network3", Subnet: "172.16.0.0/24"})

	// Find overlapping subnets
	conflicts, err := storage.FindOverlappingSubnets(context.Background())
	if err != nil {
		t.Fatalf("FindOverlappingSubnets failed: %v", err)
	}

	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for non-overlapping subnets, got %d", len(conflicts))
	}
}

func TestConflictOperations_GetConflictsByDeviceID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4"},
		},
	}
	storage.CreateDevice(context.Background(), device)

	// Create conflict involving this device
	conflict := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Test conflict",
		IPAddress:   "192.168.1.100",
		DeviceIDs:   []string{device.ID, "other-device"},
	}
	storage.CreateConflict(context.Background(), conflict)

	// Get conflicts by device ID
	conflicts, err := storage.GetConflictsByDeviceID(device.ID)
	if err != nil {
		t.Fatalf("GetConflictsByDeviceID failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
}

func TestConflictOperations_GetConflictsByDeviceIDInvalid(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetConflictsByDeviceID("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestConflictOperations_GetConflictsByIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create conflict with specific IP
	conflict := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Test conflict",
		IPAddress:   "10.0.0.100",
		DeviceIDs:   []string{"device1", "device2"},
	}
	storage.CreateConflict(context.Background(), conflict)

	// Get conflicts by IP
	conflicts, err := storage.GetConflictsByIP("10.0.0.100")
	if err != nil {
		t.Fatalf("GetConflictsByIP failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}

	if len(conflicts) > 0 && conflicts[0].IPAddress != "10.0.0.100" {
		t.Errorf("expected IP 10.0.0.100, got %s", conflicts[0].IPAddress)
	}
}

func TestConflictOperations_GetConflictsByIPInvalid(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetConflictsByIP("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestConflictOperations_MarkConflictsResolvedForDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create device
	device := &model.Device{
		Name: "server1",
		Addresses: []model.Address{
			{IP: "192.168.1.100", Type: "ipv4"},
		},
	}
	storage.CreateDevice(context.Background(), device)

	// Create active conflicts involving this device
	conflict1 := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Active conflict",
		IPAddress:   "192.168.1.100",
		DeviceIDs:   []string{device.ID},
	}
	conflict2 := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusResolved,
		Description: "Already resolved",
		IPAddress:   "192.168.1.101",
		DeviceIDs:   []string{device.ID},
	}
	storage.CreateConflict(context.Background(), conflict1)
	storage.CreateConflict(context.Background(), conflict2)

	// Mark conflicts as resolved
	err := storage.MarkConflictsResolvedForDevice(context.Background(), device.ID, "admin")
	if err != nil {
		t.Fatalf("MarkConflictsResolvedForDevice failed: %v", err)
	}

	// Verify conflict1 is now resolved
	retrieved1, _ := storage.GetConflict(conflict1.ID)
	if retrieved1.Status != model.ConflictStatusResolved {
		t.Errorf("expected conflict1 to be resolved, got %s", retrieved1.Status)
	}
	if retrieved1.ResolvedBy != "admin" {
		t.Errorf("expected resolved_by 'admin', got %s", retrieved1.ResolvedBy)
	}

	// Verify conflict2 status hasn't changed
	retrieved2, _ := storage.GetConflict(conflict2.ID)
	if retrieved2.Status != model.ConflictStatusResolved {
		t.Errorf("conflict2 should still be resolved")
	}
}

func TestConflictOperations_CreateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateConflict(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil conflict")
	}
}

func TestConflictOperations_ListOrderedByDetectedAt(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create conflicts - they should be ordered by detected_at DESC
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "First conflict",
		IPAddress:   "10.0.0.1",
	})
	storage.CreateConflict(context.Background(), &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "Second conflict",
		IPAddress:   "10.0.0.2",
	})

	conflicts, err := storage.ListConflicts(nil)
	if err != nil {
		t.Fatalf("ListConflicts failed: %v", err)
	}

	if len(conflicts) != 2 {
		t.Errorf("expected 2 conflicts, got %d", len(conflicts))
	}

	// Second conflict should come first (newer)
	if conflicts[0].Description != "Second conflict" {
		t.Error("conflicts should be ordered by detected_at DESC")
	}
}

func TestConflictOperations_WithResolvedTimestamp(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	conflict := &model.Conflict{
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusResolved,
		Description: "Test conflict",
		IPAddress:   "10.0.0.1",
	}

	if err := storage.CreateConflict(context.Background(), conflict); err != nil {
		t.Fatalf("CreateConflict failed: %v", err)
	}

	// Mark as resolved
	err := storage.UpdateConflictStatus(context.Background(), conflict.ID, model.ConflictStatusResolved, "admin", "Fixed")
	if err != nil {
		t.Fatalf("UpdateConflictStatus failed: %v", err)
	}

	retrieved, err := storage.GetConflict(conflict.ID)
	if err != nil {
		t.Fatalf("GetConflict failed: %v", err)
	}

	if retrieved.ResolvedAt == nil {
		t.Error("resolved_at should be set after status update")
	}
	if retrieved.ResolvedBy != "admin" {
		t.Errorf("expected resolved_by 'admin', got %s", retrieved.ResolvedBy)
	}
}
