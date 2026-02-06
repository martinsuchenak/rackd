package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Relationship Tests
// ============================================================================

func TestRelationshipCRUD(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer storage.Close()

	// Create two devices
	device1 := &model.Device{Name: "Server1"}
	device2 := &model.Device{Name: "Server2"}
	if err := storage.CreateDevice(context.Background(), device1); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}
	if err := storage.CreateDevice(context.Background(), device2); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Add relationship
	if err := storage.AddRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains, ""); err != nil {
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
	if err := storage.RemoveRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains); err != nil {
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
		if err := storage.CreateDevice(context.Background(), d); err != nil {
			t.Fatalf("CreateDevice failed: %v", err)
		}
	}

	// Add relationships
	storage.AddRelationship(context.Background(), parent.ID, child1.ID, model.RelationshipContains, "")
	storage.AddRelationship(context.Background(), parent.ID, child2.ID, model.RelationshipConnectedTo, "")

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
	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)

	// Add same relationship twice - should not error
	if err := storage.AddRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains, ""); err != nil {
		t.Fatalf("first AddRelationship failed: %v", err)
	}
	if err := storage.AddRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains, ""); err != nil {
		t.Fatalf("second AddRelationship failed: %v", err)
	}

	// Should still have only one relationship
	rels, _ := storage.GetRelationships(device1.ID)
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rels))
	}
}

func TestRelationshipInvalidIDs(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create valid devices for testing
	device1 := &model.Device{Name: "D1"}
	device2 := &model.Device{Name: "D2"}
	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)

	// Test with non-existent device IDs (FK constraint)
	err := storage.AddRelationship(context.Background(), "nonexistent1", "nonexistent2", model.RelationshipContains, "")
	if err == nil {
		t.Error("expected error for non-existent device IDs")
	}

	// Valid relationship should work
	err = storage.AddRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains, "")
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
	err = storage.RemoveRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains)
	if err != nil {
		t.Errorf("RemoveRelationship failed: %v", err)
	}
}

func TestRemoveRelationshipNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices but no relationship
	device1 := &model.Device{Name: "D1"}
	device2 := &model.Device{Name: "D2"}
	storage.CreateDevice(context.Background(), device1)
	storage.CreateDevice(context.Background(), device2)

	// Remove non-existent relationship should not error (idempotent)
	err := storage.RemoveRelationship(context.Background(), device1.ID, device2.ID, model.RelationshipContains)
	if err != nil {
		t.Errorf("RemoveRelationship should be idempotent, got %v", err)
	}
}

func TestGetRelatedDevicesEmpty(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	device := &model.Device{Name: "Lonely"}
	storage.CreateDevice(context.Background(), device)

	related, err := storage.GetRelatedDevices(device.ID, model.RelationshipContains)
	if err != nil {
		t.Fatalf("GetRelatedDevices failed: %v", err)
	}
	if len(related) != 0 {
		t.Errorf("expected 0 related devices, got %d", len(related))
	}
}

func TestDeleteDeviceWithRelationships(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create devices with relationships
	parent := &model.Device{Name: "Parent"}
	child := &model.Device{Name: "Child"}
	storage.CreateDevice(context.Background(), parent)
	storage.CreateDevice(context.Background(), child)
	storage.AddRelationship(context.Background(), parent.ID, child.ID, model.RelationshipContains, "")

	// Delete parent - should cascade relationships
	if err := storage.DeleteDevice(context.Background(), parent.ID); err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	// Verify relationship is gone
	rels, _ := storage.GetRelationships(child.ID)
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships after parent deletion, got %d", len(rels))
	}
}
