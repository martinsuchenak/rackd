package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
	"pgregory.net/rapid"
)

// ============================================================================
// DNS Provider Operations Tests
// ============================================================================

func TestDNSProviderOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	provider := &model.DNSProviderConfig{
		Name:        "Cloudflare DNS",
		Type:        model.DNSProviderTypeTechnitium,
		Endpoint:    "https://dns.example.com",
		Token:       "secret-token",
		Description: "Primary DNS provider",
	}

	// Create provider
	err := storage.CreateDNSProvider(context.Background(), provider)
	if err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	if provider.ID == "" {
		t.Error("provider ID should be set after creation")
	}
	if provider.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if provider.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get provider
	retrieved, err := storage.GetDNSProvider(provider.ID)
	if err != nil {
		t.Fatalf("GetDNSProvider failed: %v", err)
	}

	if retrieved.Name != provider.Name {
		t.Errorf("expected name %s, got %s", provider.Name, retrieved.Name)
	}
	if retrieved.Type != provider.Type {
		t.Errorf("expected type %s, got %s", provider.Type, retrieved.Type)
	}
	if retrieved.Endpoint != provider.Endpoint {
		t.Errorf("expected endpoint %s, got %s", provider.Endpoint, retrieved.Endpoint)
	}
	if retrieved.Token != provider.Token {
		t.Errorf("expected token %s, got %s", provider.Token, retrieved.Token)
	}
	if retrieved.Description != provider.Description {
		t.Errorf("expected description %s, got %s", provider.Description, retrieved.Description)
	}
}

func TestDNSProviderOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDNSProvider("non-existent-id")
	if err != ErrDNSProviderNotFound {
		t.Errorf("expected ErrDNSProviderNotFound, got %v", err)
	}
}

func TestDNSProviderOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:        "Original Name",
		Type:        model.DNSProviderTypeTechnitium,
		Endpoint:    "https://original.example.com",
		Token:       "original-token",
		Description: "Original description",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Update provider
	originalCreatedAt := provider.CreatedAt
	time.Sleep(10 * time.Millisecond) // Ensure updated_at is different

	provider.Name = "Updated Name"
	provider.Type = model.DNSProviderTypePowerDNS
	provider.Endpoint = "https://updated.example.com"
	provider.Token = "updated-token"
	provider.Description = "Updated description"

	err := storage.UpdateDNSProvider(context.Background(), provider)
	if err != nil {
		t.Fatalf("UpdateDNSProvider failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetDNSProvider(provider.ID)
	if err != nil {
		t.Fatalf("GetDNSProvider failed: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", retrieved.Name)
	}
	if retrieved.Type != model.DNSProviderTypePowerDNS {
		t.Errorf("expected type powerdns, got %s", retrieved.Type)
	}
	if retrieved.Endpoint != "https://updated.example.com" {
		t.Errorf("expected endpoint 'https://updated.example.com', got '%s'", retrieved.Endpoint)
	}
	if retrieved.Token != "updated-token" {
		t.Errorf("expected token 'updated-token', got '%s'", retrieved.Token)
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

func TestDNSProviderOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Delete provider
	err := storage.DeleteDNSProvider(context.Background(), provider.ID)
	if err != nil {
		t.Fatalf("DeleteDNSProvider failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetDNSProvider(provider.ID)
	if err != ErrDNSProviderNotFound {
		t.Errorf("expected ErrDNSProviderNotFound, got %v", err)
	}

	// Delete non-existent should return error
	err = storage.DeleteDNSProvider(context.Background(), "non-existent-id")
	if err != ErrDNSProviderNotFound {
		t.Errorf("expected ErrDNSProviderNotFound for non-existent, got %v", err)
	}
}

func TestDNSProviderOperations_List(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple providers
	for _, ptype := range []model.DNSProviderType{
		model.DNSProviderTypeTechnitium,
		model.DNSProviderTypePowerDNS,
		model.DNSProviderTypeBIND,
	} {
		provider := &model.DNSProviderConfig{
			Name:     string(ptype) + " Provider",
			Type:     ptype,
			Endpoint: "https://" + string(ptype) + ".example.com",
			Token:    string(ptype) + "-token",
		}
		if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
			t.Fatalf("CreateDNSProvider failed: %v", err)
		}
	}

	// List all
	providers, err := storage.ListDNSProviders(nil)
	if err != nil {
		t.Fatalf("ListDNSProviders failed: %v", err)
	}
	if len(providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(providers))
	}

	// Filter by type
	technitiumProviders, err := storage.ListDNSProviders(&model.DNSProviderFilter{
		Type: model.DNSProviderTypeTechnitium,
	})
	if err != nil {
		t.Fatalf("ListDNSProviders with type filter failed: %v", err)
	}
	if len(technitiumProviders) != 1 {
		t.Errorf("expected 1 technitium provider, got %d", len(technitiumProviders))
	}
}

func TestDNSProviderOperations_GetByName(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Unique Provider Name",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://unique.example.com",
		Token:    "unique-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Get by name
	retrieved, err := storage.GetDNSProviderByName("Unique Provider Name")
	if err != nil {
		t.Fatalf("GetDNSProviderByName failed: %v", err)
	}

	if retrieved.ID != provider.ID {
		t.Errorf("expected ID %s, got %s", provider.ID, retrieved.ID)
	}

	// Get non-existent name
	_, err = storage.GetDNSProviderByName("non-existent-name")
	if err != ErrDNSProviderNotFound {
		t.Errorf("expected ErrDNSProviderNotFound, got %v", err)
	}
}

func TestDNSProviderOperations_DeleteWithZones(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Provider With Zones",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create zone using this provider
	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Try to delete provider - should fail
	err := storage.DeleteDNSProvider(context.Background(), provider.ID)
	if err == nil {
		t.Error("expected error when deleting provider with zones")
	}
	if err != nil && err.Error() != "cannot delete DNS provider: 1 zone(s) still reference it" {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestDNSProviderOperations_AllTypes(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	types := []model.DNSProviderType{
		model.DNSProviderTypeTechnitium,
		model.DNSProviderTypePowerDNS,
		model.DNSProviderTypeBIND,
	}

	for i, ptype := range types {
		provider := &model.DNSProviderConfig{
			Name:     "Provider " + string(rune('A'+i)),
			Type:     ptype,
			Endpoint: "https://provider" + string(rune('0'+i+1)) + ".example.com",
			Token:    "token" + string(rune('0'+i+1)),
		}
		if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
			t.Fatalf("CreateDNSProvider failed for type %s: %v", ptype, err)
		}

		// Verify
		retrieved, err := storage.GetDNSProvider(provider.ID)
		if err != nil {
			t.Fatalf("GetDNSProvider failed: %v", err)
		}
		if retrieved.Type != ptype {
			t.Errorf("expected type %s, got %s", ptype, retrieved.Type)
		}
	}
}

// ============================================================================
// DNS Zone Operations Tests
// ============================================================================

func TestDNSZoneOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider first
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		AutoSync:   true,
		CreatePTR:  true,
		TTL:        3600,
		Description: "Primary zone",
	}

	// Create zone
	err := storage.CreateDNSZone(context.Background(), zone)
	if err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	if zone.ID == "" {
		t.Error("zone ID should be set after creation")
	}
	if zone.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if zone.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}
	if zone.LastSyncStatus != model.SyncStatusSuccess {
		t.Errorf("expected default sync status %s, got %s", model.SyncStatusSuccess, zone.LastSyncStatus)
	}

	// Get zone
	retrieved, err := storage.GetDNSZone(zone.ID)
	if err != nil {
		t.Fatalf("GetDNSZone failed: %v", err)
	}

	if retrieved.Name != zone.Name {
		t.Errorf("expected name %s, got %s", zone.Name, retrieved.Name)
	}
	if retrieved.ProviderID != zone.ProviderID {
		t.Errorf("expected provider_id %s, got %s", zone.ProviderID, retrieved.ProviderID)
	}
	if retrieved.AutoSync != zone.AutoSync {
		t.Errorf("expected auto_sync %v, got %v", zone.AutoSync, retrieved.AutoSync)
	}
	if retrieved.CreatePTR != zone.CreatePTR {
		t.Errorf("expected create_ptr %v, got %v", zone.CreatePTR, retrieved.CreatePTR)
	}
	if retrieved.TTL != zone.TTL {
		t.Errorf("expected ttl %d, got %d", zone.TTL, retrieved.TTL)
	}
	if retrieved.Description != zone.Description {
		t.Errorf("expected description %s, got %s", zone.Description, retrieved.Description)
	}
}

func TestDNSZoneOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDNSZone("non-existent-id")
	if err != ErrDNSZoneNotFound {
		t.Errorf("expected ErrDNSZoneNotFound, got %v", err)
	}
}

func TestDNSZoneOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create zone
	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		AutoSync:   true,
		CreatePTR:  true,
		TTL:        3600,
		Description: "Original description",
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Update zone
	originalCreatedAt := zone.CreatedAt
	time.Sleep(10 * time.Millisecond) // Ensure updated_at is different

	zone.Name = "updated.example.com"
	zone.AutoSync = false
	zone.CreatePTR = false
	zone.TTL = 7200
	zone.Description = "Updated description"

	syncTime := time.Now().UTC()
	zone.LastSyncAt = &syncTime
	zone.LastSyncStatus = model.SyncStatusFailed
	errMsg := "sync failed"
	zone.LastSyncError = &errMsg

	err := storage.UpdateDNSZone(context.Background(), zone)
	if err != nil {
		t.Fatalf("UpdateDNSZone failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetDNSZone(zone.ID)
	if err != nil {
		t.Fatalf("GetDNSZone failed: %v", err)
	}

	if retrieved.Name != "updated.example.com" {
		t.Errorf("expected name 'updated.example.com', got '%s'", retrieved.Name)
	}
	if retrieved.AutoSync != false {
		t.Errorf("expected auto_sync false, got %v", retrieved.AutoSync)
	}
	if retrieved.CreatePTR != false {
		t.Errorf("expected create_ptr false, got %v", retrieved.CreatePTR)
	}
	if retrieved.TTL != 7200 {
		t.Errorf("expected ttl 7200, got %d", retrieved.TTL)
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
	if retrieved.LastSyncStatus != model.SyncStatusFailed {
		t.Errorf("expected last_sync_status %s, got %s", model.SyncStatusFailed, retrieved.LastSyncStatus)
	}
	if retrieved.LastSyncError == nil || *retrieved.LastSyncError != errMsg {
		t.Errorf("expected last_sync_error '%s', got %v", errMsg, retrieved.LastSyncError)
	}
}

func TestDNSZoneOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create zone
	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Delete zone
	err := storage.DeleteDNSZone(context.Background(), zone.ID)
	if err != nil {
		t.Fatalf("DeleteDNSZone failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetDNSZone(zone.ID)
	if err != ErrDNSZoneNotFound {
		t.Errorf("expected ErrDNSZoneNotFound, got %v", err)
	}

	// Delete non-existent should return error
	err = storage.DeleteDNSZone(context.Background(), "non-existent-id")
	if err != ErrDNSZoneNotFound {
		t.Errorf("expected ErrDNSZoneNotFound for non-existent, got %v", err)
	}
}

func TestDNSZoneOperations_List(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create multiple zones
	for i := 1; i <= 3; i++ {
		zone := &model.DNSZone{
			Name:       "zone" + string(rune('0'+i)) + ".com",
			ProviderID: provider.ID,
			AutoSync:   i != 3, // Third one is not auto-synced
			TTL:        3600,
		}
		if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
			t.Fatalf("CreateDNSZone failed: %v", err)
		}
	}

	// List all
	zones, err := storage.ListDNSZones(nil)
	if err != nil {
		t.Fatalf("ListDNSZones failed: %v", err)
	}
	if len(zones) != 3 {
		t.Errorf("expected 3 zones, got %d", len(zones))
	}

	// Filter by provider
	providerZones, err := storage.ListDNSZones(&model.DNSZoneFilter{
		ProviderID: provider.ID,
	})
	if err != nil {
		t.Fatalf("ListDNSZones with provider filter failed: %v", err)
	}
	if len(providerZones) != 3 {
		t.Errorf("expected 3 zones for provider, got %d", len(providerZones))
	}

	// Filter by auto_sync
	autoSync := true
	autoSyncZones, err := storage.ListDNSZones(&model.DNSZoneFilter{
		AutoSync: &autoSync,
	})
	if err != nil {
		t.Fatalf("ListDNSZones with auto_sync filter failed: %v", err)
	}
	if len(autoSyncZones) != 2 {
		t.Errorf("expected 2 auto-sync zones, got %d", len(autoSyncZones))
	}
}

func TestDNSZoneOperations_GetByName(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create zone
	zone := &model.DNSZone{
		Name:       "unique.example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Get by name
	retrieved, err := storage.GetDNSZoneByName("unique.example.com")
	if err != nil {
		t.Fatalf("GetDNSZoneByName failed: %v", err)
	}

	if retrieved.ID != zone.ID {
		t.Errorf("expected ID %s, got %s", zone.ID, retrieved.ID)
	}

	// Get non-existent name
	_, err = storage.GetDNSZoneByName("non-existent.example.com")
	if err != ErrDNSZoneNotFound {
		t.Errorf("expected ErrDNSZoneNotFound, got %v", err)
	}
}

func TestDNSZoneOperations_GetByProvider(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create two providers
	provider1 := &model.DNSProviderConfig{
		Name:     "Provider 1",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://provider1.example.com",
		Token:    "token1",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider1); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	provider2 := &model.DNSProviderConfig{
		Name:     "Provider 2",
		Type:     model.DNSProviderTypePowerDNS,
		Endpoint: "https://provider2.example.com",
		Token:    "token2",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider2); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create zones for provider1
	for i := 1; i <= 2; i++ {
		zone := &model.DNSZone{
			Name:       "zone" + string(rune('0'+i)) + ".com",
			ProviderID: provider1.ID,
			TTL:        3600,
		}
		if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
			t.Fatalf("CreateDNSZone failed: %v", err)
		}
	}

	// Create zone for provider2
	zone := &model.DNSZone{
		Name:       "other.com",
		ProviderID: provider2.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Get zones by provider1
	zones, err := storage.GetDNSZonesByProvider(provider1.ID)
	if err != nil {
		t.Fatalf("GetDNSZonesByProvider failed: %v", err)
	}
	if len(zones) != 2 {
		t.Errorf("expected 2 zones for provider1, got %d", len(zones))
	}

	// Get zones by provider2
	zones, err = storage.GetDNSZonesByProvider(provider2.ID)
	if err != nil {
		t.Fatalf("GetDNSZonesByProvider failed: %v", err)
	}
	if len(zones) != 1 {
		t.Errorf("expected 1 zone for provider2, got %d", len(zones))
	}
}

func TestDNSZoneOperations_GetByNetwork(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create datacenter and network
	dc := &model.Datacenter{Name: "Test DC", Location: "Test"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	network := &model.Network{
		Name:         "Test Network",
		DatacenterID: dc.ID,
		Subnet:       "192.168.1.0/24",
	}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create zones for network
	for i := 1; i <= 2; i++ {
		zone := &model.DNSZone{
			Name:       "zone" + string(rune('0'+i)) + ".com",
			ProviderID: provider.ID,
			NetworkID:  &network.ID,
			TTL:        3600,
		}
		if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
			t.Fatalf("CreateDNSZone failed: %v", err)
		}
	}

	// Get zones by network
	zones, err := storage.GetDNSZonesByNetwork(network.ID)
	if err != nil {
		t.Fatalf("GetDNSZonesByNetwork failed: %v", err)
	}
	if len(zones) != 2 {
		t.Errorf("expected 2 zones for network, got %d", len(zones))
	}
}

func TestDNSZoneOperations_WithPTRZone(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create zone with PTR
	ptrZone := "1.168.192.in-addr.arpa"
	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		CreatePTR:  true,
		PTRZone:    &ptrZone,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Verify
	retrieved, err := storage.GetDNSZone(zone.ID)
	if err != nil {
		t.Fatalf("GetDNSZone failed: %v", err)
	}
	if retrieved.PTRZone == nil || *retrieved.PTRZone != ptrZone {
		t.Errorf("expected ptr_zone %s, got %v", ptrZone, retrieved.PTRZone)
	}
}

func TestDNSZoneOperations_WithNetwork(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	// Create datacenter and network
	dc := &model.Datacenter{Name: "Test DC", Location: "Test"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	network := &model.Network{
		Name:         "Test Network",
		DatacenterID: dc.ID,
		Subnet:       "192.168.1.0/24",
	}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Create zone with network
	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		NetworkID:  &network.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Verify
	retrieved, err := storage.GetDNSZone(zone.ID)
	if err != nil {
		t.Fatalf("GetDNSZone failed: %v", err)
	}
	if retrieved.NetworkID == nil || *retrieved.NetworkID != network.ID {
		t.Errorf("expected network_id %s, got %v", network.ID, retrieved.NetworkID)
	}
}

// ============================================================================
// DNS Record Operations Tests
// ============================================================================

func TestDNSRecordOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	record := &model.DNSRecord{
		ZoneID:     zone.ID,
		Name:       "www",
		Type:       string(model.DNSRecordTypeA),
		Value:      "192.168.1.10",
		TTL:        300,
		SyncStatus: model.RecordSyncStatusPending,
	}

	// Create record
	err := storage.CreateDNSRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	if record.ID == "" {
		t.Error("record ID should be set after creation")
	}
	if record.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if record.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}
	if record.SyncStatus != model.RecordSyncStatusPending {
		t.Errorf("expected sync_status pending, got %s", record.SyncStatus)
	}

	// Get record
	retrieved, err := storage.GetDNSRecord(record.ID)
	if err != nil {
		t.Fatalf("GetDNSRecord failed: %v", err)
	}

	if retrieved.ZoneID != record.ZoneID {
		t.Errorf("expected zone_id %s, got %s", record.ZoneID, retrieved.ZoneID)
	}
	if retrieved.Name != record.Name {
		t.Errorf("expected name %s, got %s", record.Name, retrieved.Name)
	}
	if retrieved.Type != record.Type {
		t.Errorf("expected type %s, got %s", record.Type, retrieved.Type)
	}
	if retrieved.Value != record.Value {
		t.Errorf("expected value %s, got %s", record.Value, retrieved.Value)
	}
	if retrieved.TTL != record.TTL {
		t.Errorf("expected ttl %d, got %d", record.TTL, retrieved.TTL)
	}
}

func TestDNSRecordOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDNSRecord("non-existent-id")
	if err != ErrDNSRecordNotFound {
		t.Errorf("expected ErrDNSRecordNotFound, got %v", err)
	}
}

func TestDNSRecordOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create record
	record := &model.DNSRecord{
		ZoneID:     zone.ID,
		Name:       "www",
		Type:       string(model.DNSRecordTypeA),
		Value:      "192.168.1.10",
		TTL:        300,
	}
	if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	// Update record
	originalCreatedAt := record.CreatedAt
	time.Sleep(10 * time.Millisecond) // Ensure updated_at is different

	record.Name = "mail"
	record.Type = string(model.DNSRecordTypeCNAME)
	record.Value = "example.com"
	record.TTL = 600
	record.SyncStatus = model.RecordSyncStatusSynced

	syncTime := time.Now().UTC()
	record.LastSyncAt = &syncTime
	errMsg := "sync error"
	record.ErrorMessage = &errMsg

	err := storage.UpdateDNSRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("UpdateDNSRecord failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetDNSRecord(record.ID)
	if err != nil {
		t.Fatalf("GetDNSRecord failed: %v", err)
	}

	if retrieved.Name != "mail" {
		t.Errorf("expected name 'mail', got '%s'", retrieved.Name)
	}
	if retrieved.Type != string(model.DNSRecordTypeCNAME) {
		t.Errorf("expected type CNAME, got %s", retrieved.Type)
	}
	if retrieved.Value != "example.com" {
		t.Errorf("expected value 'example.com', got '%s'", retrieved.Value)
	}
	if retrieved.TTL != 600 {
		t.Errorf("expected ttl 600, got %d", retrieved.TTL)
	}
	if retrieved.SyncStatus != model.RecordSyncStatusSynced {
		t.Errorf("expected sync_status synced, got %s", retrieved.SyncStatus)
	}
	if retrieved.ErrorMessage == nil || *retrieved.ErrorMessage != errMsg {
		t.Errorf("expected error_message '%s', got %v", errMsg, retrieved.ErrorMessage)
	}
	if retrieved.CreatedAt != originalCreatedAt {
		t.Error("created_at should not change on update")
	}
	if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
		t.Error("updated_at should be >= created_at")
	}
}

func TestDNSRecordOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create record
	record := &model.DNSRecord{
		ZoneID: zone.ID,
		Name:   "www",
		Type:   string(model.DNSRecordTypeA),
		Value:  "192.168.1.10",
		TTL:    300,
	}
	if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	// Delete record
	err := storage.DeleteDNSRecord(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("DeleteDNSRecord failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetDNSRecord(record.ID)
	if err != ErrDNSRecordNotFound {
		t.Errorf("expected ErrDNSRecordNotFound, got %v", err)
	}

	// Delete non-existent should return error
	err = storage.DeleteDNSRecord(context.Background(), "non-existent-id")
	if err != ErrDNSRecordNotFound {
		t.Errorf("expected ErrDNSRecordNotFound for non-existent, got %v", err)
	}
}

func TestDNSRecordOperations_List(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create multiple records
	for i := 1; i <= 3; i++ {
		record := &model.DNSRecord{
			ZoneID:     zone.ID,
			Name:       "record" + string(rune('0'+i)),
			Type:       string(model.DNSRecordTypeA),
			Value:      "192.168.1." + string(rune('0'+i)),
			TTL:        300,
			SyncStatus: model.RecordSyncStatusSynced,
		}
		if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("CreateDNSRecord failed: %v", err)
		}
	}

	// List all
	records, err := storage.ListDNSRecords(nil)
	if err != nil {
		t.Fatalf("ListDNSRecords failed: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	// Filter by zone
	zoneRecords, err := storage.ListDNSRecords(&model.DNSRecordFilter{
		ZoneID: zone.ID,
	})
	if err != nil {
		t.Fatalf("ListDNSRecords with zone filter failed: %v", err)
	}
	if len(zoneRecords) != 3 {
		t.Errorf("expected 3 records for zone, got %d", len(zoneRecords))
	}

	// Filter by type
	typeRecords, err := storage.ListDNSRecords(&model.DNSRecordFilter{
		Type: string(model.DNSRecordTypeA),
	})
	if err != nil {
		t.Fatalf("ListDNSRecords with type filter failed: %v", err)
	}
	if len(typeRecords) != 3 {
		t.Errorf("expected 3 A records, got %d", len(typeRecords))
	}

	// Filter by sync status
	syncStatus := model.RecordSyncStatusSynced
	syncedRecords, err := storage.ListDNSRecords(&model.DNSRecordFilter{
		SyncStatus: &syncStatus,
	})
	if err != nil {
		t.Fatalf("ListDNSRecords with sync_status filter failed: %v", err)
	}
	if len(syncedRecords) != 3 {
		t.Errorf("expected 3 synced records, got %d", len(syncedRecords))
	}
}

func TestDNSRecordOperations_GetByName(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create record
	record := &model.DNSRecord{
		ZoneID: zone.ID,
		Name:   "www",
		Type:   string(model.DNSRecordTypeA),
		Value:  "192.168.1.10",
		TTL:    300,
	}
	if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	// Get by name
	retrieved, err := storage.GetDNSRecordByName(zone.ID, "www", string(model.DNSRecordTypeA))
	if err != nil {
		t.Fatalf("GetDNSRecordByName failed: %v", err)
	}

	if retrieved.ID != record.ID {
		t.Errorf("expected ID %s, got %s", record.ID, retrieved.ID)
	}

	// Get non-existent
	_, err = storage.GetDNSRecordByName(zone.ID, "nonexistent", string(model.DNSRecordTypeA))
	if err != ErrDNSRecordNotFound {
		t.Errorf("expected ErrDNSRecordNotFound, got %v", err)
	}
}

func TestDNSRecordOperations_GetByDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

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

	// Create records for device
	for i := 1; i <= 2; i++ {
		record := &model.DNSRecord{
			ZoneID:   zone.ID,
			DeviceID: &device.ID,
			Name:     "record" + string(rune('0'+i)),
			Type:     string(model.DNSRecordTypeA),
			Value:    "192.168.1." + string(rune('0'+i)),
			TTL:      300,
		}
		if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("CreateDNSRecord failed: %v", err)
		}
	}

	// Get records by device
	records, err := storage.GetDNSRecordsByDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDNSRecordsByDevice failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records for device, got %d", len(records))
	}

	// Get records for non-existent device
	_, err = storage.GetDNSRecordsByDevice("non-existent-device-id")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDNSRecordOperations_DeleteByZone(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zones
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone1 := &model.DNSZone{
		Name:       "zone1.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone1); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	zone2 := &model.DNSZone{
		Name:       "zone2.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone2); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create records for zone1
	for i := 1; i <= 2; i++ {
		record := &model.DNSRecord{
			ZoneID: zone1.ID,
			Name:   "record" + string(rune('0'+i)),
			Type:   string(model.DNSRecordTypeA),
			Value:  "192.168.1." + string(rune('0'+i)),
			TTL:    300,
		}
		if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("CreateDNSRecord failed: %v", err)
		}
	}

	// Create record for zone2
	record := &model.DNSRecord{
		ZoneID: zone2.ID,
		Name:   "www",
		Type:   string(model.DNSRecordTypeA),
		Value:  "192.168.2.10",
		TTL:    300,
	}
	if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	// Delete records by zone1
	err := storage.DeleteDNSRecordsByZone(context.Background(), zone1.ID)
	if err != nil {
		t.Fatalf("DeleteDNSRecordsByZone failed: %v", err)
	}

	// Verify zone1 records are deleted
	zone1Records, err := storage.ListDNSRecords(&model.DNSRecordFilter{ZoneID: zone1.ID})
	if err != nil {
		t.Fatalf("ListDNSRecords failed: %v", err)
	}
	if len(zone1Records) != 0 {
		t.Errorf("expected 0 records for zone1, got %d", len(zone1Records))
	}

	// Verify zone2 records still exist
	zone2Records, err := storage.ListDNSRecords(&model.DNSRecordFilter{ZoneID: zone2.ID})
	if err != nil {
		t.Fatalf("ListDNSRecords failed: %v", err)
	}
	if len(zone2Records) != 1 {
		t.Errorf("expected 1 record for zone2, got %d", len(zone2Records))
	}

	// Delete for non-existent zone
	err = storage.DeleteDNSRecordsByZone(context.Background(), "non-existent-zone-id")
	if err != ErrDNSZoneNotFound {
		t.Errorf("expected ErrDNSZoneNotFound, got %v", err)
	}
}

func TestDNSRecordOperations_DeleteByDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create datacenter and devices
	dc := &model.Datacenter{Name: "Test DC", Location: "Test"}
	if err := storage.CreateDatacenter(context.Background(), dc); err != nil {
		t.Fatalf("CreateDatacenter failed: %v", err)
	}

	device1 := &model.Device{
		Name:         "Device 1",
		DatacenterID: dc.ID,
		Status:       model.DeviceStatusActive,
	}
	if err := storage.CreateDevice(context.Background(), device1); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	device2 := &model.Device{
		Name:         "Device 2",
		DatacenterID: dc.ID,
		Status:       model.DeviceStatusActive,
	}
	if err := storage.CreateDevice(context.Background(), device2); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Create records for device1
	for i := 1; i <= 2; i++ {
		record := &model.DNSRecord{
			ZoneID:   zone.ID,
			DeviceID: &device1.ID,
			Name:     "record" + string(rune('0'+i)),
			Type:     string(model.DNSRecordTypeA),
			Value:    "192.168.1." + string(rune('0'+i)),
			TTL:      300,
		}
		if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("CreateDNSRecord failed: %v", err)
		}
	}

	// Create record for device2
	record := &model.DNSRecord{
		ZoneID:   zone.ID,
		DeviceID: &device2.ID,
		Name:     "www",
		Type:     string(model.DNSRecordTypeA),
		Value:    "192.168.2.10",
		TTL:      300,
	}
	if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	// Delete records by device1
	err := storage.DeleteDNSRecordsByDevice(context.Background(), device1.ID)
	if err != nil {
		t.Fatalf("DeleteDNSRecordsByDevice failed: %v", err)
	}

	// Verify device1 records are deleted
	device1Records, err := storage.GetDNSRecordsByDevice(device1.ID)
	if err != nil {
		t.Fatalf("GetDNSRecordsByDevice failed: %v", err)
	}
	if len(device1Records) != 0 {
		t.Errorf("expected 0 records for device1, got %d", len(device1Records))
	}

	// Verify device2 records still exist
	device2Records, err := storage.GetDNSRecordsByDevice(device2.ID)
	if err != nil {
		t.Fatalf("GetDNSRecordsByDevice failed: %v", err)
	}
	if len(device2Records) != 1 {
		t.Errorf("expected 1 record for device2, got %d", len(device2Records))
	}

	// Delete for non-existent device
	err = storage.DeleteDNSRecordsByDevice(context.Background(), "non-existent-device-id")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestDNSRecordOperations_WithDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

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

	// Create record with device
	record := &model.DNSRecord{
		ZoneID:   zone.ID,
		DeviceID: &device.ID,
		Name:     "www",
		Type:     string(model.DNSRecordTypeA),
		Value:    "192.168.1.10",
		TTL:      300,
	}
	if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateDNSRecord failed: %v", err)
	}

	// Verify
	retrieved, err := storage.GetDNSRecord(record.ID)
	if err != nil {
		t.Fatalf("GetDNSRecord failed: %v", err)
	}
	if retrieved.DeviceID == nil || *retrieved.DeviceID != device.ID {
		t.Errorf("expected device_id %s, got %v", device.ID, retrieved.DeviceID)
	}
}

func TestDNSRecordOperations_AllRecordTypes(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	recordTypes := []model.DNSRecordType{
		model.DNSRecordTypeA,
		model.DNSRecordTypeAAAA,
		model.DNSRecordTypeCNAME,
		model.DNSRecordTypeMX,
		model.DNSRecordTypeTXT,
		model.DNSRecordTypeNS,
		model.DNSRecordTypeSOA,
		model.DNSRecordTypePTR,
		model.DNSRecordTypeSRV,
	}

	for i, rtype := range recordTypes {
		record := &model.DNSRecord{
			ZoneID: zone.ID,
			Name:   "record" + string(rune('0'+i+1)),
			Type:   string(rtype),
			Value:  "test value",
			TTL:    300,
		}
		if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("CreateDNSRecord failed for type %s: %v", rtype, err)
		}

		// Verify
		retrieved, err := storage.GetDNSRecord(record.ID)
		if err != nil {
			t.Fatalf("GetDNSRecord failed: %v", err)
		}
		if retrieved.Type != string(rtype) {
			t.Errorf("expected type %s, got %s", rtype, retrieved.Type)
		}
	}
}

func TestDNSRecordOperations_ZoneDeleteCascade(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create provider and zone
	provider := &model.DNSProviderConfig{
		Name:     "Test Provider",
		Type:     model.DNSProviderTypeTechnitium,
		Endpoint: "https://test.example.com",
		Token:    "test-token",
	}
	if err := storage.CreateDNSProvider(context.Background(), provider); err != nil {
		t.Fatalf("CreateDNSProvider failed: %v", err)
	}

	zone := &model.DNSZone{
		Name:       "example.com",
		ProviderID: provider.ID,
		TTL:        3600,
	}
	if err := storage.CreateDNSZone(context.Background(), zone); err != nil {
		t.Fatalf("CreateDNSZone failed: %v", err)
	}

	// Create records
	for i := 1; i <= 3; i++ {
		record := &model.DNSRecord{
			ZoneID: zone.ID,
			Name:   "record" + string(rune('0'+i)),
			Type:   string(model.DNSRecordTypeA),
			Value:  "192.168.1." + string(rune('0'+i)),
			TTL:    300,
		}
		if err := storage.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("CreateDNSRecord failed: %v", err)
		}
	}

	// Delete zone (should cascade delete records)
	err := storage.DeleteDNSZone(context.Background(), zone.ID)
	if err != nil {
		t.Fatalf("DeleteDNSZone failed: %v", err)
	}

	// Verify all records are deleted
	records, err := storage.ListDNSRecords(&model.DNSRecordFilter{ZoneID: zone.ID})
	if err != nil {
		t.Fatalf("ListDNSRecords failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records after zone deletion, got %d", len(records))
	}
}

// ============================================================================
// Property-Based Tests: DNS Device Linking
// ============================================================================

// Feature: dns-device-linking, Property 2: DNSRecord AddressID storage round-trip
// Validates: Requirements 2.2, 2.3
func TestDNSRecordAddressID_StorageRoundTrip(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()

		// Create a DNS provider and zone (required foreign keys)
		provider := &model.DNSProviderConfig{
			Name:     "test-provider",
			Type:     model.DNSProviderTypeTechnitium,
			Endpoint: "https://dns.example.com",
			Token:    "test-token",
		}
		if err := store.CreateDNSProvider(ctx, provider); err != nil {
			rt.Fatalf("CreateDNSProvider failed: %v", err)
		}

		zone := &model.DNSZone{
			Name:       rapid.StringMatching(`^[a-z]{3,8}\.com$`).Draw(rt, "zoneName"),
			ProviderID: provider.ID,
			TTL:        3600,
		}
		if err := store.CreateDNSZone(ctx, zone); err != nil {
			rt.Fatalf("CreateDNSZone failed: %v", err)
		}

		// Create a device with an address to get a valid address ID
		device := &model.Device{
			Name: "test-device",
			Addresses: []model.Address{
				{
					IP:   rapid.StringMatching(`^10\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`).Draw(rt, "ip"),
					Type: "ipv4",
				},
			},
			Tags:    []string{},
			Domains: []string{},
		}
		if err := store.CreateDevice(ctx, device); err != nil {
			rt.Fatalf("CreateDevice failed: %v", err)
		}

		// Retrieve device to get the generated address ID
		retrieved, err := store.GetDevice(device.ID)
		if err != nil {
			rt.Fatalf("GetDevice failed: %v", err)
		}
		if len(retrieved.Addresses) == 0 {
			rt.Fatalf("expected at least 1 address on device")
		}
		addressID := retrieved.Addresses[0].ID

		// Decide randomly whether to use a nil or non-nil AddressID
		useAddressID := rapid.Bool().Draw(rt, "useAddressID")

		var addrIDPtr *string
		if useAddressID {
			addrIDPtr = &addressID
		}

		// Create a DNS record with the chosen AddressID
		record := &model.DNSRecord{
			ZoneID:     zone.ID,
			DeviceID:   &device.ID,
			AddressID:  addrIDPtr,
			Name:       rapid.StringMatching(`^[a-z]{1,10}$`).Draw(rt, "recordName"),
			Type:       string(model.DNSRecordTypeA),
			Value:      retrieved.Addresses[0].IP,
			TTL:        300,
			SyncStatus: model.RecordSyncStatusPending,
		}
		if err := store.CreateDNSRecord(ctx, record); err != nil {
			rt.Fatalf("CreateDNSRecord failed: %v", err)
		}

		// Retrieve and verify AddressID matches (create round-trip)
		got, err := store.GetDNSRecord(record.ID)
		if err != nil {
			rt.Fatalf("GetDNSRecord failed: %v", err)
		}

		if useAddressID {
			if got.AddressID == nil {
				rt.Fatalf("expected non-nil AddressID, got nil")
			}
			if *got.AddressID != addressID {
				rt.Errorf("AddressID mismatch after create: expected %q, got %q", addressID, *got.AddressID)
			}
		} else {
			if got.AddressID != nil {
				rt.Errorf("expected nil AddressID, got %q", *got.AddressID)
			}
		}

		// Test update round-trip: flip the AddressID
		if useAddressID {
			// Was non-nil, set to nil
			record.AddressID = nil
		} else {
			// Was nil, set to non-nil
			record.AddressID = &addressID
		}
		if err := store.UpdateDNSRecord(ctx, record); err != nil {
			rt.Fatalf("UpdateDNSRecord failed: %v", err)
		}

		// Retrieve and verify the updated AddressID
		got2, err := store.GetDNSRecord(record.ID)
		if err != nil {
			rt.Fatalf("GetDNSRecord after update failed: %v", err)
		}

		if useAddressID {
			// We flipped from non-nil to nil
			if got2.AddressID != nil {
				rt.Errorf("expected nil AddressID after update, got %q", *got2.AddressID)
			}
		} else {
			// We flipped from nil to non-nil
			if got2.AddressID == nil {
				rt.Fatalf("expected non-nil AddressID after update, got nil")
			}
			if *got2.AddressID != addressID {
				rt.Errorf("AddressID mismatch after update: expected %q, got %q", addressID, *got2.AddressID)
			}
		}

		// Clean up
		_ = store.DeleteDNSRecord(ctx, record.ID)
		_ = store.DeleteDNSZone(ctx, zone.ID)
		_ = store.DeleteDevice(ctx, device.ID)
		_ = store.DeleteDNSProvider(ctx, provider.ID)
	})
}

// Feature: dns-device-linking, Property 17: Link status filter correctness
// Validates: Requirements 8.22, 8.23
func TestDNSRecordLinkStatusFilter(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	iteration := 0
	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()
		iteration++

		// Create a DNS provider and zone (required foreign keys)
		// Use unique provider name per iteration to avoid UNIQUE constraint
		provider := &model.DNSProviderConfig{
			Name:     fmt.Sprintf("test-provider-linkstatus-%d", iteration),
			Type:     model.DNSProviderTypeTechnitium,
			Endpoint: "https://dns.example.com",
			Token:    "test-token",
		}
		if err := store.CreateDNSProvider(ctx, provider); err != nil {
			rt.Fatalf("CreateDNSProvider failed: %v", err)
		}

		zone := &model.DNSZone{
			Name:       rapid.StringMatching(`^[a-z]{3,8}\.com$`).Draw(rt, "zoneName"),
			ProviderID: provider.ID,
			TTL:        3600,
		}
		if err := store.CreateDNSZone(ctx, zone); err != nil {
			rt.Fatalf("CreateDNSZone failed: %v", err)
		}

		// Create a device (needed for linked records)
		device := &model.Device{
			Name: fmt.Sprintf("test-device-linkstatus-%d", iteration),
			Addresses: []model.Address{
				{IP: "10.0.0.1", Type: "ipv4"},
			},
			Tags:    []string{},
			Domains: []string{},
		}
		if err := store.CreateDevice(ctx, device); err != nil {
			rt.Fatalf("CreateDevice failed: %v", err)
		}

		// Generate a mix of linked and unlinked records
		numLinked := rapid.IntRange(0, 5).Draw(rt, "numLinked")
		numUnlinked := rapid.IntRange(0, 5).Draw(rt, "numUnlinked")

		var linkedIDs []string
		var unlinkedIDs []string

		for i := 0; i < numLinked; i++ {
			rec := &model.DNSRecord{
				ZoneID:     zone.ID,
				DeviceID:   &device.ID,
				Name:       fmt.Sprintf("linked-%d-%d", iteration, i),
				Type:       string(model.DNSRecordTypeA),
				Value:      "10.0.0.1",
				TTL:        300,
				SyncStatus: model.RecordSyncStatusPending,
			}
			if err := store.CreateDNSRecord(ctx, rec); err != nil {
				rt.Fatalf("CreateDNSRecord (linked) failed: %v", err)
			}
			linkedIDs = append(linkedIDs, rec.ID)
		}

		for i := 0; i < numUnlinked; i++ {
			rec := &model.DNSRecord{
				ZoneID:     zone.ID,
				DeviceID:   nil,
				Name:       fmt.Sprintf("unlinked-%d-%d", iteration, i),
				Type:       string(model.DNSRecordTypeA),
				Value:      "10.0.0.2",
				TTL:        300,
				SyncStatus: model.RecordSyncStatusPending,
			}
			if err := store.CreateDNSRecord(ctx, rec); err != nil {
				rt.Fatalf("CreateDNSRecord (unlinked) failed: %v", err)
			}
			unlinkedIDs = append(unlinkedIDs, rec.ID)
		}

		// Filter by "linked" — should return exactly the linked records
		linkedStatus := "linked"
		linkedResults, err := store.ListDNSRecords(&model.DNSRecordFilter{
			ZoneID:     zone.ID,
			LinkStatus: &linkedStatus,
		})
		if err != nil {
			rt.Fatalf("ListDNSRecords (linked) failed: %v", err)
		}
		if len(linkedResults) != numLinked {
			rt.Errorf("linked filter: expected %d records, got %d", numLinked, len(linkedResults))
		}
		linkedResultIDs := make(map[string]bool)
		for _, r := range linkedResults {
			if r.DeviceID == nil {
				rt.Errorf("linked filter returned record %s with nil DeviceID", r.ID)
			}
			linkedResultIDs[r.ID] = true
		}
		for _, id := range linkedIDs {
			if !linkedResultIDs[id] {
				rt.Errorf("linked filter missing expected record %s", id)
			}
		}

		// Filter by "unlinked" — should return exactly the unlinked records
		unlinkedStatus := "unlinked"
		unlinkedResults, err := store.ListDNSRecords(&model.DNSRecordFilter{
			ZoneID:     zone.ID,
			LinkStatus: &unlinkedStatus,
		})
		if err != nil {
			rt.Fatalf("ListDNSRecords (unlinked) failed: %v", err)
		}
		if len(unlinkedResults) != numUnlinked {
			rt.Errorf("unlinked filter: expected %d records, got %d", numUnlinked, len(unlinkedResults))
		}
		unlinkedResultIDs := make(map[string]bool)
		for _, r := range unlinkedResults {
			if r.DeviceID != nil {
				rt.Errorf("unlinked filter returned record %s with non-nil DeviceID %q", r.ID, *r.DeviceID)
			}
			unlinkedResultIDs[r.ID] = true
		}
		for _, id := range unlinkedIDs {
			if !unlinkedResultIDs[id] {
				rt.Errorf("unlinked filter missing expected record %s", id)
			}
		}

		// Filter with nil LinkStatus — should return all records in the zone
		allResults, err := store.ListDNSRecords(&model.DNSRecordFilter{
			ZoneID: zone.ID,
		})
		if err != nil {
			rt.Fatalf("ListDNSRecords (all) failed: %v", err)
		}
		expectedTotal := numLinked + numUnlinked
		if len(allResults) != expectedTotal {
			rt.Errorf("nil filter: expected %d records, got %d", expectedTotal, len(allResults))
		}

		// Clean up
		for _, id := range linkedIDs {
			_ = store.DeleteDNSRecord(ctx, id)
		}
		for _, id := range unlinkedIDs {
			_ = store.DeleteDNSRecord(ctx, id)
		}
		_ = store.DeleteDNSZone(ctx, zone.ID)
		_ = store.DeleteDevice(ctx, device.ID)
		_ = store.DeleteDNSProvider(ctx, provider.ID)
	})
}
