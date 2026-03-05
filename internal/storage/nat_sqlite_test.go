package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// NAT Mapping Operations Tests
// ============================================================================

func TestNATMappingOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	mapping := &model.NATMapping{
		Name:         "Web Server NAT",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 443,
		InternalIP:   "192.168.1.10",
		InternalPort: 443,
		Protocol:     model.NATProtocolTCP,
		Description:  "HTTPS to internal web server",
		Enabled:      true,
	}

	// Create mapping
	err := storage.CreateNATMapping(context.Background(), mapping)
	if err != nil {
		t.Fatalf("CreateNATMapping failed: %v", err)
	}

	if mapping.ID == "" {
		t.Error("mapping ID should be set after creation")
	}
	if mapping.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if mapping.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get mapping
	retrieved, err := storage.GetNATMapping(context.Background(), mapping.ID)
	if err != nil {
		t.Fatalf("GetNATMapping failed: %v", err)
	}

	if retrieved.Name != mapping.Name {
		t.Errorf("expected name %s, got %s", mapping.Name, retrieved.Name)
	}
	if retrieved.ExternalIP != mapping.ExternalIP {
		t.Errorf("expected external_ip %s, got %s", mapping.ExternalIP, retrieved.ExternalIP)
	}
	if retrieved.ExternalPort != mapping.ExternalPort {
		t.Errorf("expected external_port %d, got %d", mapping.ExternalPort, retrieved.ExternalPort)
	}
	if retrieved.InternalIP != mapping.InternalIP {
		t.Errorf("expected internal_ip %s, got %s", mapping.InternalIP, retrieved.InternalIP)
	}
	if retrieved.InternalPort != mapping.InternalPort {
		t.Errorf("expected internal_port %d, got %d", mapping.InternalPort, retrieved.InternalPort)
	}
	if retrieved.Protocol != mapping.Protocol {
		t.Errorf("expected protocol %s, got %s", mapping.Protocol, retrieved.Protocol)
	}
	if retrieved.Description != mapping.Description {
		t.Errorf("expected description %s, got %s", mapping.Description, retrieved.Description)
	}
	if retrieved.Enabled != mapping.Enabled {
		t.Errorf("expected enabled %v, got %v", mapping.Enabled, retrieved.Enabled)
	}
}

func TestNATMappingOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create mapping
	mapping := &model.NATMapping{
		Name:         "Original Name",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 443,
		InternalIP:   "192.168.1.10",
		InternalPort: 443,
		Protocol:     model.NATProtocolTCP,
		Enabled:      true,
	}
	if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
		t.Fatalf("CreateNATMapping failed: %v", err)
	}

	// Update mapping
	originalCreatedAt := mapping.CreatedAt
	time.Sleep(10 * time.Millisecond) // Ensure updated_at is different

	mapping.Name = "Updated Name"
	mapping.ExternalPort = 8443
	mapping.InternalPort = 8443
	mapping.Protocol = model.NATProtocolUDP
	mapping.Enabled = false
	mapping.Description = "Updated description"

	err := storage.UpdateNATMapping(context.Background(), mapping)
	if err != nil {
		t.Fatalf("UpdateNATMapping failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetNATMapping(context.Background(), mapping.ID)
	if err != nil {
		t.Fatalf("GetNATMapping failed: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", retrieved.Name)
	}
	if retrieved.ExternalPort != 8443 {
		t.Errorf("expected external_port 8443, got %d", retrieved.ExternalPort)
	}
	if retrieved.InternalPort != 8443 {
		t.Errorf("expected internal_port 8443, got %d", retrieved.InternalPort)
	}
	if retrieved.Protocol != model.NATProtocolUDP {
		t.Errorf("expected protocol udp, got %s", retrieved.Protocol)
	}
	if retrieved.Enabled != false {
		t.Errorf("expected enabled false, got %v", retrieved.Enabled)
	}
	if retrieved.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", retrieved.Description)
	}
	if retrieved.CreatedAt != originalCreatedAt {
		t.Error("created_at should not change on update")
	}
	if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
		t.Error("updated_at should be >= created_at")
	}
}

func TestNATMappingOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create mapping
	mapping := &model.NATMapping{
		Name:         "Test NAT",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 443,
		InternalIP:   "192.168.1.10",
		InternalPort: 443,
		Protocol:     model.NATProtocolTCP,
		Enabled:      true,
	}
	if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
		t.Fatalf("CreateNATMapping failed: %v", err)
	}

	// Delete mapping
	err := storage.DeleteNATMapping(context.Background(), mapping.ID)
	if err != nil {
		t.Fatalf("DeleteNATMapping failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetNATMapping(context.Background(), mapping.ID)
	if err != ErrNATNotFound {
		t.Errorf("expected ErrNATNotFound, got %v", err)
	}

	// Delete non-existent should return error
	err = storage.DeleteNATMapping(context.Background(), "non-existent-id")
	if err != ErrNATNotFound {
		t.Errorf("expected ErrNATNotFound for non-existent, got %v", err)
	}
}

func TestNATMappingOperations_List(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple mappings
	for i := 1; i <= 3; i++ {
		mapping := &model.NATMapping{
			Name:         "NAT " + string(rune('A'+i-1)),
			ExternalIP:   "203.0.113." + string(rune('0'+i)),
			ExternalPort: 4430 + i,
			InternalIP:   "192.168.1." + string(rune('0'+i)),
			InternalPort: 4430 + i,
			Protocol:     model.NATProtocolTCP,
			Enabled:      i != 3, // Third one is disabled
		}
		if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
			t.Fatalf("CreateNATMapping failed: %v", err)
		}
	}

	// List all
	mappings, err := storage.ListNATMappings(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListNATMappings failed: %v", err)
	}
	if len(mappings) != 3 {
		t.Errorf("expected 3 mappings, got %d", len(mappings))
	}

	// Filter by enabled
	enabled := true
	enabledMappings, err := storage.ListNATMappings(context.Background(), &model.NATFilter{Enabled: &enabled})
	if err != nil {
		t.Fatalf("ListNATMappings with enabled filter failed: %v", err)
	}
	if len(enabledMappings) != 2 {
		t.Errorf("expected 2 enabled mappings, got %d", len(enabledMappings))
	}

	// Filter by protocol
	tcpMappings, err := storage.ListNATMappings(context.Background(), &model.NATFilter{Protocol: model.NATProtocolTCP})
	if err != nil {
		t.Fatalf("ListNATMappings with protocol filter failed: %v", err)
	}
	if len(tcpMappings) != 3 {
		t.Errorf("expected 3 TCP mappings, got %d", len(tcpMappings))
	}
}

func TestNATMappingOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetNATMapping(context.Background(), "non-existent-id")
	if err != ErrNATNotFound {
		t.Errorf("expected ErrNATNotFound, got %v", err)
	}
}

func TestNATMappingOperations_WithDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter and device
	dc := &model.Datacenter{Name: "Test DC", Location: "Test"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	device := &model.Device{
		Name:         "Test Device",
		DatacenterID: dc.ID,
		Status:       model.DeviceStatusActive,
	}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Create NAT mapping with device
	mapping := &model.NATMapping{
		Name:         "Device NAT",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 443,
		InternalIP:   "192.168.1.10",
		InternalPort: 443,
		Protocol:     model.NATProtocolTCP,
		DeviceID:     device.ID,
		Enabled:      true,
	}
	if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
		t.Fatalf("CreateNATMapping failed: %v", err)
	}

	// Verify device association
	retrieved, err := storage.GetNATMapping(context.Background(), mapping.ID)
	if err != nil {
		t.Fatalf("GetNATMapping failed: %v", err)
	}
	if retrieved.DeviceID != device.ID {
		t.Errorf("expected device_id %s, got %s", device.ID, retrieved.DeviceID)
	}

	// Get mappings by device
	deviceMappings, err := storage.GetNATMappingsByDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetNATMappingsByDevice failed: %v", err)
	}
	if len(deviceMappings) != 1 {
		t.Errorf("expected 1 mapping for device, got %d", len(deviceMappings))
	}
}

func TestNATMappingOperations_WithDatacenter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create datacenter
	dc := &model.Datacenter{Name: "Test DC", Location: "Test"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	// Create NAT mapping with datacenter
	mapping := &model.NATMapping{
		Name:         "DC NAT",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 443,
		InternalIP:   "192.168.1.10",
		InternalPort: 443,
		Protocol:     model.NATProtocolTCP,
		DatacenterID: dc.ID,
		Enabled:      true,
	}
	if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
		t.Fatalf("CreateNATMapping failed: %v", err)
	}

	// Get mappings by datacenter
	dcMappings, err := storage.GetNATMappingsByDatacenter(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("GetNATMappingsByDatacenter failed: %v", err)
	}
	if len(dcMappings) != 1 {
		t.Errorf("expected 1 mapping for datacenter, got %d", len(dcMappings))
	}
}

func TestNATMappingOperations_Tags(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create mapping with tags
	mapping := &model.NATMapping{
		Name:         "Tagged NAT",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 443,
		InternalIP:   "192.168.1.10",
		InternalPort: 443,
		Protocol:     model.NATProtocolTCP,
		Tags:         []string{"production", "web", "https"},
		Enabled:      true,
	}
	if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
		t.Fatalf("CreateNATMapping failed: %v", err)
	}

	// Verify tags
	retrieved, err := storage.GetNATMapping(context.Background(), mapping.ID)
	if err != nil {
		t.Fatalf("GetNATMapping failed: %v", err)
	}
	if len(retrieved.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(retrieved.Tags))
	}
	found := make(map[string]bool)
	for _, tag := range retrieved.Tags {
		found[tag] = true
	}
	if !found["production"] || !found["web"] || !found["https"] {
		t.Errorf("expected tags [production, web, https], got %v", retrieved.Tags)
	}

	// Update tags
	mapping.Tags = []string{"production", "updated"}
	if err := storage.UpdateNATMapping(context.Background(), mapping); err != nil {
		t.Fatalf("UpdateNATMapping failed: %v", err)
	}

	retrieved, err = storage.GetNATMapping(context.Background(), mapping.ID)
	if err != nil {
		t.Fatalf("GetNATMapping failed: %v", err)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags after update, got %d", len(retrieved.Tags))
	}
}

func TestNATMappingOperations_AllProtocols(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	protocols := []model.NATProtocol{model.NATProtocolTCP, model.NATProtocolUDP, model.NATProtocolAny}

	for i, protocol := range protocols {
		mapping := &model.NATMapping{
			Name:         "NAT " + string(rune('A'+i)),
			ExternalIP:   "203.0.113." + string(rune('0'+i+1)),
			ExternalPort: 443,
			InternalIP:   "192.168.1." + string(rune('0'+i+1)),
			InternalPort: 443,
			Protocol:     protocol,
			Enabled:      true,
		}
		if err := storage.CreateNATMapping(context.Background(), mapping); err != nil {
			t.Fatalf("CreateNATMapping failed for protocol %s: %v", protocol, err)
		}

		// Verify
		retrieved, err := storage.GetNATMapping(context.Background(), mapping.ID)
		if err != nil {
			t.Fatalf("GetNATMapping failed: %v", err)
		}
		if retrieved.Protocol != protocol {
			t.Errorf("expected protocol %s, got %s", protocol, retrieved.Protocol)
		}
	}
}
