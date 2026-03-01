package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DeviceService struct {
	store           storage.ExtendedStorage
	conflictService *ConflictService
	dns             *DNSService
}

func NewDeviceService(store storage.ExtendedStorage) *DeviceService {
	return &DeviceService{store: store}
}

func (s *DeviceService) setConflictService(cs *ConflictService) {
	s.conflictService = cs
}

func (s *DeviceService) setDNSService(dns *DNSService) {
	s.dns = dns
}

// boolPtr returns a pointer to the given bool value
func boolPtr(v bool) *bool {
	return &v
}

// syncDeviceDNS creates/updates DNS records for a device when it has a hostname
func (s *DeviceService) syncDeviceDNS(ctx context.Context, device *model.Device) error {
	if s.dns == nil {
		return nil
	}
	if device.Hostname == "" {
		return nil
	}

	// Find zones for this device's networks with auto_sync enabled
	zones, err := s.store.ListDNSZones(&model.DNSZoneFilter{
		AutoSync: boolPtr(true),
	})
	if err != nil {
		return err
	}

	// For each zone linked to device's network
	for _, zone := range zones {
		if zone.NetworkID != nil && deviceHasNetworkID(device, *zone.NetworkID) {
			// Create/update A record for each IP in this network
			for _, addr := range device.Addresses {
				if addr.NetworkID == *zone.NetworkID && addr.IP != "" {
					req := &model.CreateDNSRecordRequest{
						ZoneID:   zone.ID,
						DeviceID: &device.ID,
						Name:     device.Hostname,
						Type:     "A",
						Value:    addr.IP,
						TTL:      zone.TTL,
					}
					if _, err := s.dns.CreateRecord(ctx, req); err != nil {
						// Log but don't fail - individual record failures shouldn't block
						continue
					}

					// Create PTR record if enabled
					if zone.CreatePTR && zone.PTRZone != nil && *zone.PTRZone != "" {
						ptrReq := &model.CreateDNSRecordRequest{
							ZoneID:   zone.ID,
							DeviceID: &device.ID,
							Name:     extractPTRName(addr.IP),
							Type:     "PTR",
							Value:    device.Hostname + "." + zone.Name,
							TTL:      zone.TTL,
						}
						if _, err := s.dns.CreateRecord(ctx, ptrReq); err != nil {
							// Log but don't fail - individual record failures shouldn't block
							continue
						}
					}
				}
			}
		}
	}
	return nil
}

// deviceHasNetworkID checks if a device has an address in the specified network
func deviceHasNetworkID(device *model.Device, networkID string) bool {
	for _, addr := range device.Addresses {
		if addr.NetworkID == networkID {
			return true
		}
	}
	return false
}

// extractPTRName extracts a PTR record name from an IP address
func extractPTRName(ipStr string) string {
	parts := strings.Split(ipStr, ".")
	if len(parts) != 4 {
		return ""
	}
	// Reverse the IP for in-addr.arpa format
	return fmt.Sprintf("%s.%s.%s.in-addr.arpa", parts[3], parts[2], parts[1])
}

// checkForIPConflicts checks if any IP addresses are duplicates and creates conflict records
func (s *DeviceService) checkForIPConflicts(ctx context.Context, device *model.Device) {
	if s.conflictService == nil {
		return
	}

	// Check each IP address for duplicates
	for _, addr := range device.Addresses {
		if addr.IP == "" {
			continue
		}

		// Look for existing devices with this IP
		conflicts, err := s.store.GetConflictsByIP(addr.IP)
		if err != nil {
			continue
		}

		// Check if we already have a conflict for this IP
		hasConflict := false
		for _, c := range conflicts {
			if c.Status == model.ConflictStatusActive {
				hasConflict = true
				break
			}
		}

		// If no active conflict exists and we found multiple devices with this IP, create one
		if !hasConflict && len(conflicts) == 0 {
			// Find all devices with this IP (including the current one)
			allDevices, err := s.store.ListDevices(&model.DeviceFilter{})
			if err != nil {
				continue
			}

			var deviceIDs []string
			var deviceNames []string
			for _, d := range allDevices {
				for _, a := range d.Addresses {
					if a.IP == addr.IP {
						deviceIDs = append(deviceIDs, d.ID)
						deviceNames = append(deviceNames, d.Name)
						break
					}
				}
			}

			// Only create conflict if multiple devices have this IP
			if len(deviceIDs) > 1 {
				conflict := &model.Conflict{
					Type:        model.ConflictTypeDuplicateIP,
					Status:      model.ConflictStatusActive,
					Description: "IP address assigned to multiple devices",
					IPAddress:   addr.IP,
					DeviceIDs:   deviceIDs,
					DeviceNames: deviceNames,
				}
				s.conflictService.store.CreateConflict(ctx, conflict)
			}
		}
	}
}

// validateStatus validates the device status
func validateStatus(status model.DeviceStatus) error {
	if status != "" && !status.IsValid() {
		return ValidationErrors{{Field: "status", Message: "Invalid status. Must be one of: planned, active, maintenance, decommissioned"}}
	}
	return nil
}

// setStatusChangedBy sets the StatusChangedBy field from the context
func setStatusChangedBy(ctx context.Context, device *model.Device) {
	caller := CallerFrom(ctx)
	if caller != nil && caller.UserID != "" {
		device.StatusChangedBy = caller.UserID
	}
}

func (s *DeviceService) List(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDevices(filter)
}

func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "create"); err != nil {
		return err
	}

	if device.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	// Validate status
	if err := validateStatus(device.Status); err != nil {
		return err
	}

	// Set status changed by from context
	setStatusChangedBy(ctx, device)

	err := s.store.CreateDevice(enrichAuditCtx(ctx), device)
	if err != nil {
		return err
	}

	// Check for IP conflicts after creation
	s.checkForIPConflicts(ctx, device)

	// Sync DNS records if device has a hostname
	if err := s.syncDeviceDNS(ctx, device); err != nil {
		// Log but don't fail - DNS sync failures shouldn't block device creation
	}

	return nil
}

func (s *DeviceService) Get(ctx context.Context, id string) (*model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "read"); err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(id)
	if err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return device, nil
}

func (s *DeviceService) Update(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return err
	}

	if device.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if device.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	// Validate status
	if err := validateStatus(device.Status); err != nil {
		return err
	}

	// Set status changed by from context
	setStatusChangedBy(ctx, device)

	err := s.store.UpdateDevice(enrichAuditCtx(ctx), device)
	if err != nil {
		return err
	}

	// Check for IP conflicts after update
	s.checkForIPConflicts(ctx, device)

	// Sync DNS records if device has a hostname
	if err := s.syncDeviceDNS(ctx, device); err != nil {
		// Log but don't fail - DNS sync failures shouldn't block device update
	}

	return nil
}

func (s *DeviceService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "devices", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDevice(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DeviceService) Search(ctx context.Context, query string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "search"); err != nil {
		return nil, err
	}

	return s.store.SearchDevices(query)
}

// GetStatusCounts returns the count of devices by status
func (s *DeviceService) GetStatusCounts(ctx context.Context) (map[model.DeviceStatus]int, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}

	return s.store.GetDeviceStatusCounts()
}
