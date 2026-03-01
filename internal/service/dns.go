package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/dns"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

var (
	// ErrDNSProviderInUse is returned when trying to delete a provider that has zones
	ErrDNSProviderInUse = fmt.Errorf("dns provider is in use")
	// ErrDNSZoneInUse is returned when trying to delete a zone that has records
	ErrDNSZoneInUse = fmt.Errorf("dns zone is in use")
	// ErrDNSProviderNotFound is returned when a provider is not found
	ErrDNSProviderNotFound = fmt.Errorf("dns provider not found")
	// ErrDNSZoneNotFound is returned when a zone is not found
	ErrDNSZoneNotFound = fmt.Errorf("dns zone not found")
	// ErrDNSRecordNotFound is returned when a record is not found
	ErrDNSRecordNotFound = fmt.Errorf("dns record not found")
)

// DNSService handles DNS provider, zone, and record operations
type DNSService struct {
	store         storage.ExtendedStorage
	encryptor     *credentials.Encryptor
	providerCache map[string]dns.Provider
	mu            sync.RWMutex
}

// NewDNSService creates a new DNS service instance
func NewDNSService(store storage.ExtendedStorage, encryptor *credentials.Encryptor) *DNSService {
	return &DNSService{
		store:         store,
		encryptor:     encryptor,
		providerCache: make(map[string]dns.Provider),
	}
}

// Provider CRUD Operations

// CreateProvider creates a new DNS provider configuration
func (s *DNSService) CreateProvider(ctx context.Context, req *model.CreateDNSProviderRequest) (*model.DNSProviderConfig, error) {
	if err := requirePermission(ctx, s.store, "dns-provider", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.Type == "" || !req.Type.IsValid() {
		return nil, ValidationErrors{{Field: "type", Message: "Type must be one of: technitium, powerdns, bind"}}
	}
	if req.Endpoint == "" {
		return nil, ValidationErrors{{Field: "endpoint", Message: "Endpoint is required"}}
	}
	if req.Token == "" {
		return nil, ValidationErrors{{Field: "token", Message: "Token is required"}}
	}

	// Check if provider with same name already exists
	if _, err := s.store.GetDNSProviderByName(req.Name); err == nil {
		return nil, ErrAlreadyExists
	}

	// Encrypt the token
	encryptedToken, err := s.encryptor.Encrypt(req.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	provider := &model.DNSProviderConfig{
		Name:        req.Name,
		Type:        req.Type,
		Endpoint:    req.Endpoint,
		Token:       encryptedToken,
		Description: req.Description,
	}

	if err := s.store.CreateDNSProvider(ctx, provider); err != nil {
		return nil, err
	}

	// Don't return the token in the response
	provider.Token = ""
	return provider, nil
}

// GetProvider returns a DNS provider by ID (without token)
func (s *DNSService) GetProvider(ctx context.Context, id string) (*model.DNSProviderConfig, error) {
	if err := requirePermission(ctx, s.store, "dns-provider", "read"); err != nil {
		return nil, err
	}

	provider, err := s.store.GetDNSProvider(id)
	if err != nil {
		if err == storage.ErrDNSProviderNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Don't expose the token
	provider.Token = ""
	return provider, nil
}

// ListProviders returns all DNS providers with optional filtering
func (s *DNSService) ListProviders(ctx context.Context, filter *model.DNSProviderFilter) ([]model.DNSProviderConfig, error) {
	if err := requirePermission(ctx, s.store, "dns-provider", "list"); err != nil {
		return nil, err
	}

	providers, err := s.store.ListDNSProviders(filter)
	if err != nil {
		return nil, err
	}

	// Clear tokens from response
	for i := range providers {
		providers[i].Token = ""
	}

	return providers, nil
}

// UpdateProvider updates an existing DNS provider
func (s *DNSService) UpdateProvider(ctx context.Context, id string, req *model.UpdateDNSProviderRequest) (*model.DNSProviderConfig, error) {
	if err := requirePermission(ctx, s.store, "dns-provider", "update"); err != nil {
		return nil, err
	}

	provider, err := s.store.GetDNSProvider(id)
	if err != nil {
		if err == storage.ErrDNSProviderNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		// Check if new name conflicts with existing provider
		if existing, err := s.store.GetDNSProviderByName(*req.Name); err == nil && existing.ID != id {
			return nil, ErrAlreadyExists
		}
		provider.Name = *req.Name
	}
	if req.Type != nil {
		if !req.Type.IsValid() {
			return nil, ValidationErrors{{Field: "type", Message: "Invalid provider type"}}
		}
		provider.Type = *req.Type
	}
	if req.Endpoint != nil {
		if *req.Endpoint == "" {
			return nil, ValidationErrors{{Field: "endpoint", Message: "Endpoint cannot be empty"}}
		}
		provider.Endpoint = *req.Endpoint
	}
	if req.Token != nil {
		if *req.Token == "" {
			return nil, ValidationErrors{{Field: "token", Message: "Token cannot be empty"}}
		}
		encryptedToken, err := s.encryptor.Encrypt(*req.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		provider.Token = encryptedToken
		// Invalidate cached provider since token changed
		s.mu.Lock()
		delete(s.providerCache, id)
		s.mu.Unlock()
	}
	if req.Description != nil {
		provider.Description = *req.Description
	}

	if err := s.store.UpdateDNSProvider(ctx, provider); err != nil {
		return nil, err
	}

	// Don't return the token
	provider.Token = ""
	return provider, nil
}

// DeleteProvider deletes a DNS provider
func (s *DNSService) DeleteProvider(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "dns-provider", "delete"); err != nil {
		return err
	}

	// Check if provider has any zones
	zones, err := s.store.GetDNSZonesByProvider(id)
	if err != nil {
		return err
	}
	if len(zones) > 0 {
		return ErrDNSProviderInUse
	}

	if err := s.store.DeleteDNSProvider(ctx, id); err != nil {
		if err == storage.ErrDNSProviderNotFound {
			return ErrNotFound
		}
		return err
	}

	// Remove from cache
	s.mu.Lock()
	delete(s.providerCache, id)
	s.mu.Unlock()

	return nil
}

// TestProvider tests the connectivity to a DNS provider
func (s *DNSService) TestProvider(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "dns-provider", "test"); err != nil {
		return err
	}

	provider, err := s.getProvider(ctx, id)
	if err != nil {
		return err
	}

	return provider.HealthCheck(ctx)
}

// Zone CRUD Operations

// CreateZone creates a new DNS zone
func (s *DNSService) CreateZone(ctx context.Context, req *model.CreateDNSZoneRequest) (*model.DNSZone, error) {
	if err := requirePermission(ctx, s.store, "dns-zone", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.ProviderID == "" {
		return nil, ValidationErrors{{Field: "provider_id", Message: "Provider ID is required"}}
	}

	// Validate provider exists
	if _, err := s.store.GetDNSProvider(req.ProviderID); err != nil {
		if err == storage.ErrDNSProviderNotFound {
			return nil, ValidationErrors{{Field: "provider_id", Message: "Provider not found"}}
		}
		return nil, err
	}

	// Check if zone with same name already exists
	if _, err := s.store.GetDNSZoneByName(req.Name); err == nil {
		return nil, ErrAlreadyExists
	}

	// Validate network if provided
	if req.NetworkID != nil && *req.NetworkID != "" {
		if _, err := s.store.GetNetwork(*req.NetworkID); err != nil {
			return nil, ValidationErrors{{Field: "network_id", Message: "Network not found"}}
		}
	}

	// Set default TTL
	if req.TTL <= 0 {
		req.TTL = 3600
	}

	zone := &model.DNSZone{
		Name:           req.Name,
		ProviderID:     req.ProviderID,
		NetworkID:      req.NetworkID,
		AutoSync:       req.AutoSync,
		CreatePTR:      req.CreatePTR,
		PTRZone:        req.PTRZone,
		TTL:            req.TTL,
		Description:    req.Description,
		LastSyncStatus: model.SyncStatusSuccess,
	}

	if err := s.store.CreateDNSZone(ctx, zone); err != nil {
		return nil, err
	}

	return zone, nil
}

// GetZone returns a DNS zone by ID
func (s *DNSService) GetZone(ctx context.Context, id string) (*model.DNSZone, error) {
	if err := requirePermission(ctx, s.store, "dns-zone", "read"); err != nil {
		return nil, err
	}

	zone, err := s.store.GetDNSZone(id)
	if err != nil {
		if err == storage.ErrDNSZoneNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return zone, nil
}

// ListZones returns all DNS zones with optional filtering
func (s *DNSService) ListZones(ctx context.Context, filter *model.DNSZoneFilter) ([]model.DNSZone, error) {
	if err := requirePermission(ctx, s.store, "dns-zone", "list"); err != nil {
		return nil, err
	}

	return s.store.ListDNSZones(filter)
}

// UpdateZone updates an existing DNS zone
func (s *DNSService) UpdateZone(ctx context.Context, id string, req *model.UpdateDNSZoneRequest) (*model.DNSZone, error) {
	if err := requirePermission(ctx, s.store, "dns-zone", "update"); err != nil {
		return nil, err
	}

	zone, err := s.store.GetDNSZone(id)
	if err != nil {
		if err == storage.ErrDNSZoneNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		// Check if new name conflicts with existing zone
		if existing, err := s.store.GetDNSZoneByName(*req.Name); err == nil && existing.ID != id {
			return nil, ErrAlreadyExists
		}
		zone.Name = *req.Name
	}
	if req.NetworkID != nil {
		if *req.NetworkID != "" {
			if _, err := s.store.GetNetwork(*req.NetworkID); err != nil {
				return nil, ValidationErrors{{Field: "network_id", Message: "Network not found"}}
			}
		}
		zone.NetworkID = req.NetworkID
	}
	if req.AutoSync != nil {
		zone.AutoSync = *req.AutoSync
	}
	if req.CreatePTR != nil {
		zone.CreatePTR = *req.CreatePTR
	}
	if req.PTRZone != nil {
		zone.PTRZone = req.PTRZone
	}
	if req.TTL != nil {
		if *req.TTL <= 0 {
			return nil, ValidationErrors{{Field: "ttl", Message: "TTL must be positive"}}
		}
		zone.TTL = *req.TTL
	}
	if req.Description != nil {
		zone.Description = *req.Description
	}

	if err := s.store.UpdateDNSZone(ctx, zone); err != nil {
		return nil, err
	}

	return zone, nil
}

// DeleteZone deletes a DNS zone and all its records
func (s *DNSService) DeleteZone(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "dns-zone", "delete"); err != nil {
		return err
	}

	// Check if zone exists
	zone, err := s.store.GetDNSZone(id)
	if err != nil {
		if err == storage.ErrDNSZoneNotFound {
			return ErrNotFound
		}
		return err
	}

	// Delete all records in the zone
	if err := s.store.DeleteDNSRecordsByZone(ctx, id); err != nil {
		return fmt.Errorf("failed to delete zone records: %w", err)
	}

	// Delete the zone
	if err := s.store.DeleteDNSZone(ctx, id); err != nil {
		return err
	}

	// If the zone is configured for PTR and has a PTR zone, delete it too
	if zone.PTRZone != nil {
		if ptrZone, err := s.store.GetDNSZoneByName(*zone.PTRZone); err == nil {
			// Only delete if it's owned by the same provider
			if ptrZone.ProviderID == zone.ProviderID {
				s.store.DeleteDNSRecordsByZone(ctx, ptrZone.ID)
				s.store.DeleteDNSZone(ctx, ptrZone.ID)
			}
		}
	}

	return nil
}

// Record CRUD Operations

// CreateRecord creates a new DNS record
func (s *DNSService) CreateRecord(ctx context.Context, req *model.CreateDNSRecordRequest) (*model.DNSRecord, error) {
	if err := requirePermission(ctx, s.store, "dns", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.ZoneID == "" {
		return nil, ValidationErrors{{Field: "zone_id", Message: "Zone ID is required"}}
	}
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.Type == "" {
		return nil, ValidationErrors{{Field: "type", Message: "Type is required"}}
	}
	if req.Value == "" {
		return nil, ValidationErrors{{Field: "value", Message: "Value is required"}}
	}

	// Validate zone exists
	zone, err := s.store.GetDNSZone(req.ZoneID)
	if err != nil {
		if err == storage.ErrDNSZoneNotFound {
			return nil, ValidationErrors{{Field: "zone_id", Message: "Zone not found"}}
		}
		return nil, err
	}

	// Validate device if provided
	if req.DeviceID != nil && *req.DeviceID != "" {
		if _, err := s.store.GetDevice(*req.DeviceID); err != nil {
			return nil, ValidationErrors{{Field: "device_id", Message: "Device not found"}}
		}
	}

	// Check for duplicate record
	if existing, err := s.store.GetDNSRecordByName(req.ZoneID, req.Name, req.Type); err == nil {
		// Record exists, update it instead
		return s.UpdateRecord(ctx, existing.ID, &model.UpdateDNSRecordRequest{
			DeviceID: req.DeviceID,
			Value:    &req.Value,
			TTL:      &req.TTL,
		})
	}

	// Set default TTL
	ttl := req.TTL
	if ttl <= 0 {
		ttl = zone.TTL
	}

	record := &model.DNSRecord{
		ZoneID:     req.ZoneID,
		DeviceID:   req.DeviceID,
		Name:       req.Name,
		Type:       req.Type,
		Value:      req.Value,
		TTL:        ttl,
		SyncStatus: model.RecordSyncStatusPending,
	}

	if err := s.store.CreateDNSRecord(ctx, record); err != nil {
		return nil, err
	}

	// Auto-sync if zone is configured for it
	if zone.AutoSync {
		if err := s.SyncRecord(ctx, record); err != nil {
			// Update record with error
			errMsg := err.Error()
			record.ErrorMessage = &errMsg
			record.SyncStatus = model.RecordSyncStatusFailed
			s.store.UpdateDNSRecord(ctx, record)
		}
	}

	return record, nil
}

// GetRecord returns a DNS record by ID
func (s *DNSService) GetRecord(ctx context.Context, id string) (*model.DNSRecord, error) {
	if err := requirePermission(ctx, s.store, "dns", "read"); err != nil {
		return nil, err
	}

	record, err := s.store.GetDNSRecord(id)
	if err != nil {
		if err == storage.ErrDNSRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return record, nil
}

// ListRecords returns all DNS records with optional filtering
func (s *DNSService) ListRecords(ctx context.Context, filter *model.DNSRecordFilter) ([]model.DNSRecord, error) {
	if err := requirePermission(ctx, s.store, "dns", "list"); err != nil {
		return nil, err
	}

	return s.store.ListDNSRecords(filter)
}

// UpdateRecord updates an existing DNS record
func (s *DNSService) UpdateRecord(ctx context.Context, id string, req *model.UpdateDNSRecordRequest) (*model.DNSRecord, error) {
	if err := requirePermission(ctx, s.store, "dns", "update"); err != nil {
		return nil, err
	}

	record, err := s.store.GetDNSRecord(id)
	if err != nil {
		if err == storage.ErrDNSRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Validate device if provided
	if req.DeviceID != nil && *req.DeviceID != "" {
		if _, err := s.store.GetDevice(*req.DeviceID); err != nil {
			return nil, ValidationErrors{{Field: "device_id", Message: "Device not found"}}
		}
	}

	// Apply updates
	updated := false
	if req.DeviceID != nil {
		record.DeviceID = req.DeviceID
		updated = true
	}
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		record.Name = *req.Name
		updated = true
	}
	if req.Type != nil {
		if *req.Type == "" {
			return nil, ValidationErrors{{Field: "type", Message: "Type cannot be empty"}}
		}
		record.Type = *req.Type
		updated = true
	}
	if req.Value != nil {
		if *req.Value == "" {
			return nil, ValidationErrors{{Field: "value", Message: "Value cannot be empty"}}
		}
		record.Value = *req.Value
		updated = true
	}
	if req.TTL != nil {
		if *req.TTL < 0 {
			return nil, ValidationErrors{{Field: "ttl", Message: "TTL cannot be negative"}}
		}
		record.TTL = *req.TTL
		updated = true
	}

	if updated {
		record.SyncStatus = model.RecordSyncStatusPending
		record.ErrorMessage = nil
	}

	if err := s.store.UpdateDNSRecord(ctx, record); err != nil {
		return nil, err
	}

	// Auto-sync if zone is configured for it
	if zone, err := s.store.GetDNSZone(record.ZoneID); err == nil && zone.AutoSync && updated {
		if err := s.SyncRecord(ctx, record); err != nil {
			// Update record with error
			errMsg := err.Error()
			record.ErrorMessage = &errMsg
			record.SyncStatus = model.RecordSyncStatusFailed
			s.store.UpdateDNSRecord(ctx, record)
		}
	}

	return record, nil
}

// DeleteRecord deletes a DNS record
func (s *DNSService) DeleteRecord(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "dns", "delete"); err != nil {
		return err
	}

	record, err := s.store.GetDNSRecord(id)
	if err != nil {
		if err == storage.ErrDNSRecordNotFound {
			return ErrNotFound
		}
		return err
	}

	// Get zone for provider info
	zone, err := s.store.GetDNSZone(record.ZoneID)
	if err != nil {
		return err
	}

	// Try to delete from DNS provider if synced
	if record.SyncStatus == model.RecordSyncStatusSynced {
		if provider, err := s.getProvider(ctx, zone.ProviderID); err == nil {
			if err := provider.DeleteRecord(ctx, zone.Name, record.Name, record.Type); err != nil {
				// Log but don't fail - the record may not exist on the provider
			}
		}
	}

	if err := s.store.DeleteDNSRecord(ctx, id); err != nil {
		return err
	}

	return nil
}

// Sync Operations

// SyncZone syncs all pending records in a zone to the DNS provider
func (s *DNSService) SyncZone(ctx context.Context, zoneID string) (*model.SyncResult, error) {
	if err := requirePermission(ctx, s.store, "dns-zone", "sync"); err != nil {
		return nil, err
	}

	zone, err := s.store.GetDNSZone(zoneID)
	if err != nil {
		if err == storage.ErrDNSZoneNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Verify provider exists and is accessible
	if _, err := s.getProvider(ctx, zone.ProviderID); err != nil {
		return &model.SyncResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Get all records in the zone
	records, err := s.store.ListDNSRecords(&model.DNSRecordFilter{ZoneID: zoneID})
	if err != nil {
		return nil, err
	}

	result := &model.SyncResult{
		Total:     len(records),
		FailedIDs: []string{},
	}

	now := time.Now()

	for _, record := range records {
		if err := s.SyncRecord(ctx, &record); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, record.ID)
		} else {
			result.Synced++
		}
	}

	// Update zone sync status
	if result.Failed == 0 {
		zone.LastSyncStatus = model.SyncStatusSuccess
		zone.LastSyncError = nil
		result.Success = true
	} else if result.Synced == 0 {
		zone.LastSyncStatus = model.SyncStatusFailed
		errMsg := fmt.Sprintf("All %d records failed to sync", result.Failed)
		zone.LastSyncError = &errMsg
		result.Error = errMsg
	} else {
		zone.LastSyncStatus = model.SyncStatusPartial
		errMsg := fmt.Sprintf("%d of %d records failed to sync", result.Failed, result.Total)
		zone.LastSyncError = &errMsg
		result.Error = errMsg
	}
	zone.LastSyncAt = &now

	s.store.UpdateDNSZone(ctx, zone)

	return result, nil
}

// SyncDevice syncs all DNS records associated with a device
func (s *DNSService) SyncDevice(ctx context.Context, deviceID string) (*model.SyncResult, error) {
	if err := requirePermission(ctx, s.store, "dns", "sync"); err != nil {
		return nil, err
	}

	// Get all records for this device
	records, err := s.store.GetDNSRecordsByDevice(deviceID)
	if err != nil {
		return nil, err
	}

	result := &model.SyncResult{
		Total:     len(records),
		FailedIDs: []string{},
	}

	for _, record := range records {
		if err := s.SyncRecord(ctx, &record); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, record.ID)
		} else {
			result.Synced++
		}
	}

	result.Success = result.Failed == 0

	return result, nil
}

// ImportFromDNS imports all records from a DNS provider zone
func (s *DNSService) ImportFromDNS(ctx context.Context, zoneID string) (*model.ImportResult, error) {
	if err := requirePermission(ctx, s.store, "dns-zone", "import"); err != nil {
		return nil, err
	}

	zone, err := s.store.GetDNSZone(zoneID)
	if err != nil {
		if err == storage.ErrDNSZoneNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	provider, err := s.getProvider(ctx, zone.ProviderID)
	if err != nil {
		return &model.ImportResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Get records from provider
	dnsRecords, err := provider.ListRecords(ctx, zone.Name)
	if err != nil {
		return &model.ImportResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	result := &model.ImportResult{
		Total:      len(dnsRecords),
		SkippedIDs: []string{},
		FailedIDs:  []string{},
	}

	for _, dnsRecord := range dnsRecords {
		// Check if record already exists
		existing, err := s.store.GetDNSRecordByName(zoneID, dnsRecord.Name, dnsRecord.Type)
		if err == nil {
			// Record exists, update value if different
			if existing.Value != dnsRecord.Value {
				existing.Value = dnsRecord.Value
				if dnsRecord.TTL > 0 {
					existing.TTL = dnsRecord.TTL
				}
				existing.SyncStatus = model.RecordSyncStatusSynced
				if err := s.store.UpdateDNSRecord(ctx, existing); err != nil {
					result.Failed++
					result.FailedIDs = append(result.FailedIDs, dnsRecord.Name)
				} else {
					result.Imported++
				}
			} else {
				result.Skipped++
				result.SkippedIDs = append(result.SkippedIDs, dnsRecord.Name)
			}
		} else if err == storage.ErrDNSRecordNotFound {
			// Create new record
			record := &model.DNSRecord{
				ZoneID:     zoneID,
				Name:       dnsRecord.Name,
				Type:       dnsRecord.Type,
				Value:      dnsRecord.Value,
				TTL:        dnsRecord.TTL,
				SyncStatus: model.RecordSyncStatusSynced,
			}
			if dnsRecord.TTL == 0 {
				record.TTL = zone.TTL
			}
			if err := s.store.CreateDNSRecord(ctx, record); err != nil {
				result.Failed++
				result.FailedIDs = append(result.FailedIDs, dnsRecord.Name)
			} else {
				result.Imported++
			}
		} else {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, dnsRecord.Name)
		}
	}

	result.Success = result.Failed == 0

	return result, nil
}

// Helper Functions

// getProvider returns a cached or newly created DNS provider client
func (s *DNSService) getProvider(ctx context.Context, providerID string) (dns.Provider, error) {
	// Try cache first
	s.mu.RLock()
	if provider, ok := s.providerCache[providerID]; ok {
		s.mu.RUnlock()
		return provider, nil
	}
	s.mu.RUnlock()

	// Get provider config from storage
	config, err := s.store.GetDNSProvider(providerID)
	if err != nil {
		return nil, ErrDNSProviderNotFound
	}

	// Decrypt token
	token, err := s.encryptor.Decrypt(config.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt provider token: %w", err)
	}

	// Create provider client based on type
	var provider dns.Provider
	switch config.Type {
	case model.DNSProviderTypeTechnitium:
		provider = dns.NewTechnitiumClient(config.Endpoint, token)
	case model.DNSProviderTypePowerDNS:
		return nil, fmt.Errorf("powerdns provider not yet implemented")
	case model.DNSProviderTypeBIND:
		return nil, fmt.Errorf("bind provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}

	// Cache the provider
	s.mu.Lock()
	s.providerCache[providerID] = provider
	s.mu.Unlock()

	return provider, nil
}

// SyncRecord syncs a single record to the DNS provider
func (s *DNSService) SyncRecord(ctx context.Context, record *model.DNSRecord) error {
	zone, err := s.store.GetDNSZone(record.ZoneID)
	if err != nil {
		return err
	}

	provider, err := s.getProvider(ctx, zone.ProviderID)
	if err != nil {
		return err
	}

	dnsRecord := &dns.Record{
		Name:  record.Name,
		Type:  record.Type,
		Value: record.Value,
		TTL:   record.TTL,
	}

	// Check if record exists on provider
	existing, err := provider.GetRecord(ctx, zone.Name, record.Name, record.Type)
	if err == nil {
		// Record exists, update it
		if existing.Value != record.Value {
			if err := provider.UpdateRecord(ctx, zone.Name, dnsRecord); err != nil {
				return err
			}
		}
	} else {
		// Record doesn't exist, create it
		if err := provider.CreateRecord(ctx, zone.Name, dnsRecord); err != nil {
			return err
		}
	}

	// Update record as synced
	now := time.Now()
	record.SyncStatus = model.RecordSyncStatusSynced
	record.LastSyncAt = &now
	record.ErrorMessage = nil

	return s.store.UpdateDNSRecord(ctx, record)
}

// generatePTRZone generates a PTR zone name from a CIDR subnet
func (s *DNSService) generatePTRZone(subnet string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}

	// For IPv4
	if ip.To4() != nil {
		ones, _ := ipnet.Mask.Size()
		if ones < 24 {
			return "", fmt.Errorf("CIDR must be at least /24 for PTR zone generation")
		}

		// Convert to in-addr.arpa format
		reversed := reverseIP(ip.String())
		parts := strings.Split(reversed, ".")

		// Determine how many octets to include based on mask size
		var octets int
		switch {
		case ones >= 24:
			octets = 3
		case ones >= 16:
			octets = 2
		default:
			octets = 1
		}

		return strings.Join(parts[0:octets], ".") + ".in-addr.arpa.", nil
	}

	// For IPv6
	if ip.To16() != nil {
		// Convert to ip6.arpa format (nibble reversal)
		reversed := reverseIPv6(ip.String())
		maskSize, _ := ipnet.Mask.Size()

		// Determine how many nibbles to include
		nibbles := maskSize / 4
		if nibbles > len(reversed) {
			nibbles = len(reversed)
		}

		return reversed[:nibbles] + ".ip6.arpa.", nil
	}

	return "", fmt.Errorf("unsupported IP address format")
}

// extractPTRName extracts a PTR record name from an IP address
func (s *DNSService) extractPTRName(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	// For IPv4
	if ip.To4() != nil {
		return reverseIP(ip.String()) + ".in-addr.arpa."
	}

	// For IPv6
	if ip.To16() != nil {
		return reverseIPv6(ip.String()) + ".ip6.arpa."
	}

	return ""
}

// reverseIP reverses an IPv4 address (e.g., "192.168.1.1" -> "1.1.168.192")
func reverseIP(ip string) string {
	parts := strings.Split(ip, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

// reverseIPv6 reverses an IPv6 address into nibble notation
func reverseIPv6(ip string) string {
	// Parse and expand IPv6 address
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}

	// Get 16-byte representation
	bytes := parsed.To16()
	if bytes == nil {
		return ""
	}

	// Convert to hex nibbles (reversed order)
	var nibbles []string
	for i := 15; i >= 0; i-- {
		b := bytes[i]
		nibbles = append(nibbles, fmt.Sprintf("%x", b&0x0f))
		nibbles = append(nibbles, fmt.Sprintf("%x", (b>>4)&0x0f))
	}

	return strings.Join(nibbles, ".")
}
